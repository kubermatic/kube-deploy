package secret

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
	"k8s.io/kube-deploy/cluster-api-kubeception/pkg/resources/namespace"
	clusterv1alpha1 "k8s.io/kube-deploy/cluster-api/api/cluster/v1alpha1"
)

const (
	rootCASecretName = "root-ca"

	rootCASecretCertKey = "ca.crt"
	rootCASecretKeyKey  = "ca.key"
)

func EnsureClusterCASecretExists(cluster *clusterv1alpha1.Cluster, lister listerscorev1.SecretLister, client kubernetes.Interface) error {
	ns := namespace.ClusterNamespaceName(cluster)

	exists, err := secretExists(rootCASecretName, ns, lister)
	if err != nil {
		return err
	}

	if !exists {
		k, err := triple.NewCA(fmt.Sprintf("root-ca.%s", cluster.Name))
		if err != nil {
			return fmt.Errorf("failed to create root-ca: %v", err)
		}

		_, err = client.CoreV1().Secrets(ns).Create(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: rootCASecretName,
			},
			Data: map[string][]byte{
				rootCASecretCertKey: cert.EncodeCertPEM(k.Cert),
				rootCASecretKeyKey:  cert.EncodePrivateKeyPEM(k.Key),
			},
			Type: corev1.SecretTypeOpaque,
		})

		return err
	}

	return nil
}
