package service

import (
	"fmt"
	"net"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/kube-deploy/cluster-api-kubeception/pkg/errors"
	"k8s.io/kube-deploy/cluster-api-kubeception/pkg/resources/namespace"
	clusterv1alpha1 "k8s.io/kube-deploy/cluster-api/api/cluster/v1alpha1"
)

const (
	externalApiserverServiceName = "apiserver-external"
)

func ExternalApiserverLoadBalancerIPGetter(lister listerscorev1.ServiceLister) func(*clusterv1alpha1.Cluster) (net.IP, error) {
	return LoadBalancerIPGetter(externalApiserverServiceName, lister)
}

func LoadBalancerIPGetter(name string, lister listerscorev1.ServiceLister) func(*clusterv1alpha1.Cluster) (net.IP, error) {
	return func(cluster *clusterv1alpha1.Cluster) (net.IP, error) {
		ns := namespace.ClusterNamespaceName(cluster)
		s, err := lister.Services(ns).Get(name)
		if err != nil {
			return nil, fmt.Errorf("failed to get service %s/%s from lister: %v", ns, name, err)
		}
		for _, i := range s.Status.LoadBalancer.Ingress {
			if i.IP != "" {
				ip := net.ParseIP(i.IP)
				if ip == nil {
					return nil, fmt.Errorf("failed to parse lb ip %s", i.IP)
				}
				return ip, nil
			}
		}
		return nil, errors.NoIPAvailableYetErr
	}
}

func EnsureExternalApiserverServiceExists(cluster *clusterv1alpha1.Cluster, lister listerscorev1.ServiceLister, client kubernetes.Interface) error {
	ns := namespace.ClusterNamespaceName(cluster)

	exists, err := serviceExists(externalApiserverServiceName, ns, lister)
	if err != nil {
		return err
	}

	if !exists {
		_, err = client.CoreV1().Services(ns).Create(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: externalApiserverServiceName,
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeLoadBalancer,
				Ports: []corev1.ServicePort{
					{
						Name:       "apiserver-tls",
						Port:       6443,
						TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: 6443},
						Protocol:   corev1.ProtocolTCP,
					},
				},
				Selector: map[string]string{"foo": "bar"},
			},
		})
		return err
	}
	return nil
}
