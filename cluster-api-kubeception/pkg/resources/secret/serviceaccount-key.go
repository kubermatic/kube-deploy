package secret

import (
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/kube-deploy/cluster-api-kubeception/pkg/resources/namespace"
	clusterv1alpha1 "k8s.io/kube-deploy/cluster-api/api/cluster/v1alpha1"
)

const (
	serviceAccountKeySecretName = "service-account-key"

	serviceAccountKeyKey = "serviceaccount.key"
)

func EnsureClusterServiceAccountkeySecretExists(cluster *clusterv1alpha1.Cluster, lister listerscorev1.SecretLister, client kubernetes.Interface) error {
	ns := namespace.ClusterNamespaceName(cluster)

	exists, err := secretExists(serviceAccountKeySecretName, ns, lister)
	if err != nil {
		return err
	}

	if !exists {
		priv, err := rsa.GenerateKey(cryptorand.Reader, 4096)
		if err != nil {
			return fmt.Errorf("failed to generate a private key: %v", err)
		}

		saKey := x509.MarshalPKCS1PrivateKey(priv)
		block := pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: saKey,
		}

		_, err = client.CoreV1().Secrets(ns).Create(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: serviceAccountKeySecretName,
			},
			Data: map[string][]byte{
				serviceAccountKeyKey: pem.EncodeToMemory(&block),
			},
			Type: corev1.SecretTypeOpaque,
		})

		return err
	}

	return nil
}
