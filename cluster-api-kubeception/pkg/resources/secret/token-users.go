package secret

import (
	"bytes"
	"encoding/csv"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/kube-deploy/cluster-api-kubeception/pkg/resources/namespace"
	clusterv1alpha1 "k8s.io/kube-deploy/cluster-api/api/cluster/v1alpha1"
)

const (
	tokenUsersSecretName = "token-users"

	tokenUsersSecretKey = "token-users.csv"
)

func EnsureClusterTokenUsersSecretExists(cluster *clusterv1alpha1.Cluster, lister listerscorev1.SecretLister, client kubernetes.Interface) error {
	ns := namespace.ClusterNamespaceName(cluster)

	exists, err := secretExists(tokenUsersSecretName, ns, lister)
	if err != nil {
		return err
	}

	if !exists {
		adminToken := fmt.Sprintf("%s.%s", rand.String(6), rand.String(16))

		buffer := bytes.Buffer{}
		writer := csv.NewWriter(&buffer)
		if err := writer.Write([]string{adminToken, "admin", "10000", "system:masters"}); err != nil {
			return err
		}
		writer.Flush()

		_, err = client.CoreV1().Secrets(ns).Create(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: tokenUsersSecretName,
			},
			Data: map[string][]byte{
				tokenUsersSecretKey: buffer.Bytes(),
			},
			Type: corev1.SecretTypeOpaque,
		})

		return err
	}

	return nil
}
