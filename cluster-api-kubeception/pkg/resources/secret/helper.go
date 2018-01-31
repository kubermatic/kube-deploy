package secret

import (
	"fmt"
	"net"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	clusterv1alpha1 "k8s.io/kube-deploy/cluster-api/api/cluster/v1alpha1"
)

type ExternalIPRetriever func(*clusterv1alpha1.Cluster) (net.IP, error)

func secretExists(name, namespace string, lister listerscorev1.SecretLister) (bool, error) {
	_, err := lister.Secrets(namespace).Get(name)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("failed to get secret %q from lister: %v", name, err)
	}
	return true, nil
}

// stolen from https://github.com/gardener/gardener/blob/master/pkg/operation/common/utils.go#L92
// ComputeClusterIP parses the provided <cidr> and sets the last byte to the value of <lastByte>.
// For example, <cidr> = 100.64.0.0/11 and <lastByte> = 10 the result would be 100.64.0.10
func ComputeClusterIP(cidr string, lastByte byte) net.IP {
	ip, _, _ := net.ParseCIDR(cidr)
	ip = ip.To4()
	ip[3] = lastByte
	return ip
}
