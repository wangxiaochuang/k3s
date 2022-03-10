package cluster

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/k3s-io/kine/pkg/client"
	"github.com/k3s-io/kine/pkg/endpoint"
	"github.com/sirupsen/logrus"
	"github.com/wangxiaochuang/k3s/pkg/bootstrap"
	"github.com/wangxiaochuang/k3s/pkg/clientaccess"
	"github.com/wangxiaochuang/k3s/pkg/daemons/config"
)

func Save(ctx context.Context, config *config.Control, etcdConfig endpoint.ETCDConfig, override bool) error {
	buf := &bytes.Buffer{}
	if err := bootstrap.ReadFromDisk(buf, &config.Runtime.ControlRuntimeBootstrap); err != nil {
		return err
	}
	token := config.Token
	if token == "" {
		tokenFromFile, err := readTokenFromFile(config.Runtime.ServerToken, config.Runtime.ServerCA, config.DataDir)
		if err != nil {
			return err
		}
		token = tokenFromFile
	}
	normalizedToken, err := normalizeToken(token)
	if err != nil {
		return err
	}

	data, err := encrypt(normalizedToken, buf.Bytes())
	if err != nil {
		return err
	}

	storageClient, err := client.New(etcdConfig)
	if err != nil {
		return err
	}
	defer storageClient.Close()

	if _, _, err = getBootstrapKeyFromStorage(ctx, storageClient, normalizedToken, token); err != nil {
		return err
	}

	if err := storageClient.Create(ctx, storageKey(normalizedToken), data); err != nil {
		if err.Error() == "key exists" {
			logrus.Warn("bootstrap key already exists")
			if override {
				bsd, err := bootstrapKeyData(ctx, storageClient)
				if err != nil {
					return err
				}
				return storageClient.Update(ctx, storageKey(normalizedToken), bsd.Modified, data)
			}
			return nil
		} else if strings.Contains(err.Error(), "not supported for learner") {
			logrus.Debug("skipping bootstrap data save on learner")
			return nil
		}
		return err
	}

	return nil
}

func bootstrapKeyData(ctx context.Context, storageClient client.Client) (*client.Value, error) {
	bootstrapList, err := storageClient.List(ctx, "/bootstrap", 0)
	if err != nil {
		return nil, err
	}
	if len(bootstrapList) == 0 {
		return nil, errors.New("no bootstrap data found")
	}
	if len(bootstrapList) > 1 {
		return nil, errors.New("found multiple bootstrap keys in storage")
	}
	return &bootstrapList[0], nil
}

func (c *Cluster) storageBootstrap(ctx context.Context) error {
	if err := c.startStorage(ctx); err != nil {
		return err
	}

	storageClient, err := client.New(c.EtcdConfig)
	if err != nil {
		return err
	}
	defer storageClient.Close()

	token := c.config.Token
	if token == "" {
		tokenFromFile, err := readTokenFromFile(c.runtime.ServerToken, c.runtime.ServerCA, c.config.DataDir)
		if err != nil {
			return err
		}
		if tokenFromFile == "" {
			// at this point this is a fresh start in a non managed environment
			c.saveBootstrap = true
			return nil
		}
		token = tokenFromFile
	}
	panic("in storageBootstrap")
}

func getBootstrapKeyFromStorage(ctx context.Context, storageClient client.Client, normalizedToken, oldToken string) (*client.Value, bool, error) {
	emptyStringKey := storageKey("")
	tokenKey := storageKey(normalizedToken)
	bootstrapList, err := storageClient.List(ctx, "/bootstrap", 0)
	if err != nil {
		return nil, false, err
	}
	if len(bootstrapList) == 0 {
		return nil, true, nil
	}
	if len(bootstrapList) > 1 {
		logrus.Warn("found multiple bootstrap keys in storage")
	}
	// check for empty string key and for old token format with k10 prefix
	if err := migrateOldTokens(ctx, bootstrapList, storageClient, emptyStringKey, tokenKey, normalizedToken, oldToken); err != nil {
		return nil, false, err
	}

	// getting the list of bootstrap again after migrating the empty key
	bootstrapList, err = storageClient.List(ctx, "/bootstrap", 0)
	if err != nil {
		return nil, false, err
	}
	for _, bootstrapKV := range bootstrapList {
		// ensure bootstrap is stored in the current token's key
		if string(bootstrapKV.Key) == tokenKey {
			return &bootstrapKV, false, nil
		}
	}

	return nil, false, errors.New("bootstrap data already found and encrypted with different token")
}

func readTokenFromFile(serverToken, certs, dataDir string) (string, error) {
	tokenFile := filepath.Join(dataDir, "token")

	b, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		if os.IsNotExist(err) {
			token, err := clientaccess.FormatToken(serverToken, certs)
			if err != nil {
				return token, err
			}
			return token, nil
		}
		return "", err
	}

	// strip the token from any new line if its read from file
	return string(bytes.TrimRight(b, "\n")), nil
}

func normalizeToken(token string) (string, error) {
	_, password, ok := clientaccess.ParseUsernamePassword(token)
	if !ok {
		return password, errors.New("failed to normalize token; must be in format K10<CA-HASH>::<USERNAME>:<PASSWORD> or <PASSWORD>")
	}

	return password, nil
}

func migrateOldTokens(ctx context.Context, bootstrapList []client.Value, storageClient client.Client, emptyStringKey, tokenKey, token, oldToken string) error {
	oldTokenKey := storageKey(oldToken)

	for _, bootstrapKV := range bootstrapList {
		// checking for empty string bootstrap key
		if string(bootstrapKV.Key) == emptyStringKey {
			logrus.Warn("bootstrap data encrypted with empty string, deleting and resaving with token")
			if err := doMigrateToken(ctx, storageClient, bootstrapKV, "", emptyStringKey, token, tokenKey); err != nil {
				return err
			}
		} else if string(bootstrapKV.Key) == oldTokenKey && oldTokenKey != tokenKey {
			logrus.Warn("bootstrap data encrypted with old token format string, deleting and resaving with token")
			if err := doMigrateToken(ctx, storageClient, bootstrapKV, oldToken, oldTokenKey, token, tokenKey); err != nil {
				return err
			}
		}
	}

	return nil
}

func doMigrateToken(ctx context.Context, storageClient client.Client, keyValue client.Value, oldToken, oldTokenKey, newToken, newTokenKey string) error {
	// make sure that the process is non-destructive by decrypting/re-encrypting/storing the data before deleting the old key
	data, err := decrypt(oldToken, keyValue.Data)
	if err != nil {
		return err
	}

	encryptedData, err := encrypt(newToken, data)
	if err != nil {
		return err
	}

	// saving the new encrypted data with the right token key
	if err := storageClient.Create(ctx, newTokenKey, encryptedData); err != nil {
		if err.Error() == "key exists" {
			logrus.Warn("bootstrap key exists")
		} else if strings.Contains(err.Error(), "not supported for learner") {
			logrus.Debug("skipping bootstrap data save on learner")
			return nil
		} else {
			return err
		}
	}

	logrus.Infof("created bootstrap key %s", newTokenKey)
	// deleting the old key
	if err := storageClient.Delete(ctx, oldTokenKey, keyValue.Modified); err != nil {
		logrus.Warnf("failed to delete old bootstrap key %s", oldTokenKey)
	}

	return nil
}
