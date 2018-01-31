package secret

import (
	"crypto/rsa"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
	"k8s.io/kube-deploy/cluster-api-kubeception/pkg/resources/namespace"
	clusterv1alpha1 "k8s.io/kube-deploy/cluster-api/api/cluster/v1alpha1"
)

const (
	apiserverKubeletClientCertSecretName = "apiserver-kubelet-client-certs"

	kubeletClientCertCertKey = "kubelet.crt"
	kubeletClientCertKeyKey  = "kubelet.key"
)

func EnsureClusterApiserverKubeletCertSecretExists(cluster *clusterv1alpha1.Cluster, lister listerscorev1.SecretLister, client kubernetes.Interface, retrieveIP ExternalIPRetriever) error {
	ns := namespace.ClusterNamespaceName(cluster)

	exists, err := secretExists(apiserverKubeletClientCertSecretName, ns, lister)
	if err != nil {
		return err
	}

	if !exists {
		externalIP, err := retrieveIP(cluster)
		if err != nil {
			return err
		}

		//get root ca
		caSecret, err := lister.Secrets(ns).Get(rootCASecretName)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return fmt.Errorf("no root ca exist so far to create the apiserver tls certificate")
			}
			return err
		}

		caCerts, err := cert.ParseCertsPEM(caSecret.Data[rootCASecretCertKey])
		if err != nil {
			return fmt.Errorf("failed to parse ca cert: %v", err)
		}

		caKey, err := cert.ParsePrivateKeyPEM(caSecret.Data[rootCASecretKeyKey])
		if err != nil {
			return fmt.Errorf("failed to parse ca key: %v", err)
		}

		caKp := &triple.KeyPair{
			Cert: caCerts[0],
			Key:  caKey.(*rsa.PrivateKey),
		}

		kubeletKp, err := triple.NewClientKeyPair(caKp, externalIP.String(), []string{externalIP.String()})
		if err != nil {

			return fmt.Errorf("failed to create apiserver kubelet client key pair: %v", err)
		}

		_, err = client.CoreV1().Secrets(ns).Create(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: apiserverKubeletClientCertSecretName,
			},
			Data: map[string][]byte{
				kubeletClientCertCertKey: cert.EncodeCertPEM(kubeletKp.Cert),
				kubeletClientCertKeyKey:  cert.EncodePrivateKeyPEM(kubeletKp.Key),
			},
			Type: corev1.SecretTypeOpaque,
		})

		return err
	}

	return nil
}
