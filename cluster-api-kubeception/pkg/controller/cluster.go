/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	v1beta2appslisters "k8s.io/client-go/listers/apps/v1beta2"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/kube-deploy/cluster-api-kubeception/pkg/resources/namespace"
	"k8s.io/kube-deploy/cluster-api-kubeception/pkg/resources/secret"
	"k8s.io/kube-deploy/cluster-api-kubeception/pkg/resources/service"
	clusterv1alpha1 "k8s.io/kube-deploy/cluster-api/api/cluster/v1alpha1"
	clusterapiclient "k8s.io/kube-deploy/cluster-api/client/clientset/versioned"
	clusterapilisters "k8s.io/kube-deploy/cluster-api/client/listers/cluster/v1alpha1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"
)

const (
	emptyJSONPatch = "{}"

	workerPeriod = 500 * time.Millisecond
)

type Controller struct {
	kubeClient    kubernetes.Interface
	clusterClient clusterapiclient.Interface

	clusterLister        clusterapilisters.ClusterLister
	namespaceLister      corev1listers.NamespaceLister
	serviceLister        corev1listers.ServiceLister
	secretLister         corev1listers.SecretLister
	configMapLister      corev1listers.ConfigMapLister
	serviceAccountLister corev1listers.ServiceAccountLister
	deploymentLister     v1beta2appslisters.DeploymentLister

	queue workqueue.RateLimitingInterface
}

func NewClusterController(
	queue workqueue.RateLimitingInterface,
	kubeClient kubernetes.Interface,
	clusterClient clusterapiclient.Interface,
	clusterLister clusterapilisters.ClusterLister,
	namespaceLister corev1listers.NamespaceLister,
	serviceLister corev1listers.ServiceLister,
	secretLister corev1listers.SecretLister,
	configMapLister corev1listers.ConfigMapLister,
	serviceAccountLister corev1listers.ServiceAccountLister,
	deploymentLister v1beta2appslisters.DeploymentLister) *Controller {

	controller := &Controller{
		kubeClient:    kubeClient,
		clusterClient: clusterClient,

		clusterLister:        clusterLister,
		namespaceLister:      namespaceLister,
		serviceLister:        serviceLister,
		secretLister:         secretLister,
		configMapLister:      configMapLister,
		serviceAccountLister: serviceAccountLister,
		deploymentLister:     deploymentLister,

		queue: queue,
	}

	return controller
}

func (c *Controller) Run(threads int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.queue.ShutDown()

	for i := 0; i < threads; i++ {
		go wait.Until(c.runWorker, workerPeriod, stopCh)
	}

	<-stopCh
	return nil
}

func (c *Controller) runWorker() {
	c.processNextWorkItem()
}

func (c *Controller) processNextWorkItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}

	defer c.queue.Done(key)

	glog.V(6).Infof("Processing cluster: %s", key)
	err := c.syncHandler(key.(string))
	if err == nil {
		c.queue.Forget(key)
		return true
	}

	utilruntime.HandleError(fmt.Errorf("%v failed with: %v", key, err))
	c.queue.AddRateLimited(key)

	return true
}

func (c *Controller) ensureClusterResourcesExist(cluster *clusterv1alpha1.Cluster) error {
	if err := namespace.EnsureClusterNamespaceExists(cluster, c.namespaceLister, c.kubeClient); err != nil {
		return fmt.Errorf("failed to ensure that the cluster namespace exists: %v", err)
	}

	if err := service.EnsureExternalApiserverServiceExists(cluster, c.serviceLister, c.kubeClient); err != nil {
		return fmt.Errorf("failed to ensure that the external apiserver service exists: %v", err)
	}

	if err := secret.EnsureClusterCASecretExists(cluster, c.secretLister, c.kubeClient); err != nil {
		return fmt.Errorf("failed to ensure that the cluster root ca secret exists: %v", err)
	}

	if err := secret.EnsureClusterApiserverTLSCertSecretExists(cluster, c.secretLister, c.kubeClient, service.ExternalApiserverLoadBalancerIPGetter(c.serviceLister)); err != nil {
		return fmt.Errorf("failed to ensure that the apiserver tls cert secret exists: %v", err)
	}

	if err := secret.EnsureClusterApiserverKubeletCertSecretExists(cluster, c.secretLister, c.kubeClient, service.ExternalApiserverLoadBalancerIPGetter(c.serviceLister)); err != nil {
		return fmt.Errorf("failed to ensure that the apiserver kubelet client cert secret exists: %v", err)
	}

	if err := secret.EnsureClusterServiceAccountkeySecretExists(cluster, c.secretLister, c.kubeClient); err != nil {
		return fmt.Errorf("failed to ensure that the service account key secret exists: %v", err)
	}

	if err := secret.EnsureClusterTokenUsersSecretExists(cluster, c.secretLister, c.kubeClient); err != nil {
		return fmt.Errorf("failed to ensure that the token users csv secret exists: %v", err)
	}

	return nil
}

func (c *Controller) syncHandler(key string) error {
	listerCluster, err := c.clusterLister.Get(key)
	if err != nil {
		if kerrors.IsNotFound(err) {
			glog.V(6).Infof("cluster '%s' in work queue no longer exists", key)
			return nil
		}
		return err
	}
	cluster := listerCluster.DeepCopy()

	return c.ensureClusterResourcesExist(cluster)
}

func (c *Controller) patchCluster(newCluster, oldCluster *clusterv1alpha1.Cluster) error {
	currentCluster, err := c.clusterClient.ClusterV1alpha1().Clusters().Get(newCluster.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get current cluster from lister: %v", err)
	}
	currentJSON, err := json.Marshal(currentCluster)
	if err != nil {
		return fmt.Errorf("failed to marshal current cluster to json: %v", err)
	}
	oldJSON, err := json.Marshal(oldCluster)
	if err != nil {
		return fmt.Errorf("failed to marshal old cluster to json: %v", err)
	}
	newJSON, err := json.Marshal(newCluster)
	if err != nil {
		return fmt.Errorf("failed to marshal updated cluster to json: %v", err)
	}
	patch, err := jsonmergepatch.CreateThreeWayJSONMergePatch(oldJSON, newJSON, currentJSON)
	if err != nil {
		return fmt.Errorf("failed to create three-way-json-merge-patch: %v", err)
	}
	if string(patch) == emptyJSONPatch {
		//nothing to do
		return nil
	}
	_, err = c.clusterClient.ClusterV1alpha1().Clusters().Patch(newCluster.Name, types.MergePatchType, patch)
	return err
}
