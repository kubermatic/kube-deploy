package namespace

import (
	"fmt"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	clusterv1alpha1 "k8s.io/kube-deploy/cluster-api/api/cluster/v1alpha1"
)

const (
	clusterNamespacePrefix = "cluster-"
)

func ClusterNamespaceName(cluster *clusterv1alpha1.Cluster) string {
	return clusterNamespacePrefix + cluster.Name
}

func namespaceExists(name string, lister listerscorev1.NamespaceLister) (bool, error) {
	_, err := lister.Get(name)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("failed to get namespace %q from lister: %v", name, err)
	}
	return true, nil
}
