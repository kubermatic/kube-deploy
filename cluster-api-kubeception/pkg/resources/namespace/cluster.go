package namespace

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	clusterv1alpha1 "k8s.io/kube-deploy/cluster-api/api/cluster/v1alpha1"
)

func EnsureClusterNamespaceExists(cluster *clusterv1alpha1.Cluster, lister listerscorev1.NamespaceLister, client kubernetes.Interface) error {
	name := ClusterNamespaceName(cluster)

	exists, err := namespaceExists(name, lister)
	if err != nil {
		return err
	}

	if !exists {
		_, err = client.CoreV1().Namespaces().Create(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		})
		return err
	}
	return nil
}
