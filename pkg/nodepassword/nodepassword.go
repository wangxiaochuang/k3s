package nodepassword

import (
	"strings"

	coreclient "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/wangxiaochuang/k3s/pkg/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getSecretName(nodeName string) string {
	return strings.ToLower(nodeName + ".node-password." + version.Program)
}

func Delete(secretClient coreclient.SecretClient, nodeName string) error {
	return secretClient.Delete(metav1.NamespaceSystem, getSecretName(nodeName), &metav1.DeleteOptions{})
}
