package cluster

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
)

func (c *Cluster) Bootstrap(ctx context.Context, snapshot bool) error {
	if err := c.assignManagedDriver(ctx); err != nil {
		return err
	}

	// true false nil
	shouldBootstrap, isInitialized, err := c.shouldBootstrapLoad(ctx)
	if err != nil {
		return err
	}
	c.shouldBootstrap = shouldBootstrap

	if c.managedDB != nil {
		panic(fmt.Sprintf("in Bootstrap %+v", isInitialized))
	}

	if c.shouldBootstrap {
		return c.bootstrap(ctx)
	}

	return nil
}

func copyFile(src, dst string) error {
	srcfd, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcfd.Close()

	dstfd, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstfd.Close()

	if _, err = io.Copy(dstfd, srcfd); err != nil {
		return err
	}

	srcinfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcinfo.Mode())
}

// createTmpDataDir creates a temporary directory and copies the
// contents of the original etcd data dir to be used
// by etcd when reading data.
func createTmpDataDir(src, dst string) error {
	srcinfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcinfo.Mode()); err != nil {
		return err
	}

	fds, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, fd := range fds {
		srcfp := path.Join(src, fd.Name())
		dstfp := path.Join(dst, fd.Name())

		if fd.IsDir() {
			if err = createTmpDataDir(srcfp, dstfp); err != nil {
				fmt.Println(err)
			}
		} else {
			if err = copyFile(srcfp, dstfp); err != nil {
				fmt.Println(err)
			}
		}
	}

	return nil
}

func (c *Cluster) shouldBootstrapLoad(ctx context.Context) (bool, bool, error) {
	// Non-nil managedDB indicates that the database is either initialized, initializing, or joining
	if c.managedDB != nil {
		panic("in shouldBootstrapLoad")
	}

	return true, false, nil
}

func isDirEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}

	return false, err
}

// certDirsExist

func (c *Cluster) bootstrap(ctx context.Context) error {
	c.joining = true

	if c.runtime.HTTPBootstrap {
		panic("in bootstrap")
	}

	return c.storageBootstrap(ctx)
}
