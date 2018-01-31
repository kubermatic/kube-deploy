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

package main

import (
	"context"
	"flag"
	"time"

	"github.com/golang/glog"
	"github.com/juju/ratelimit"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/kube-deploy/cluster-api-kubeception/pkg/controller"
	clusterv1alpha1 "k8s.io/kube-deploy/cluster-api/api/cluster/v1alpha1"
	clusterapiclient "k8s.io/kube-deploy/cluster-api/client/clientset/versioned"
	clusterapiinformers "k8s.io/kube-deploy/cluster-api/client/informers/externalversions"
)

var (
	masterURL   string
	kubeconfig  string
	workerCount int
)

const (
	informerResyncPeriod = time.Minute * 5
)

func main() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.IntVar(&workerCount, "worker-count", 5, "Number of workers to process machines. Using a high number with a lot of machines might cause getting rate-limited from your cloud provider.")

	flag.Parse()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		glog.Fatalf("Error building kubeconfig: %v", err)
	}

	kubeClient := kubernetes.NewForConfigOrDie(cfg)
	clusterClient := clusterapiclient.NewForConfigOrDie(cfg)
	extClient := apiextensionsclient.NewForConfigOrDie(cfg)

	_, err = clusterv1alpha1.CreateMachinesCRD(extClient)
	if err != nil && !kerrors.IsAlreadyExists(err) {
		glog.Fatalf("Error creating Machines CRD: %v\n", err)
	}
	_, err = clusterv1alpha1.CreateClustersCRD(extClient)
	if err != nil && !kerrors.IsAlreadyExists(err) {
		glog.Fatalf("Error creating Clusters CRD: %v\n", err)
	}

	rateLimiter := workqueue.NewMaxOfRateLimiter(
		workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 10*time.Second),
		&workqueue.BucketRateLimiter{Bucket: ratelimit.NewBucketWithRate(float64(10), int64(100))},
	)
	queue := workqueue.NewNamedRateLimitingQueue(rateLimiter, "Cluster")

	clusterAPIInformerFactory := clusterapiinformers.NewSharedInformerFactory(clusterClient, informerResyncPeriod)
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, informerResyncPeriod)

	namespaceInformer := kubeInformerFactory.Core().V1().Namespaces()
	namespaceInformer.Informer()
	serviceInformer := kubeInformerFactory.Core().V1().Services()
	serviceInformer.Informer()
	secretInformer := kubeInformerFactory.Core().V1().Secrets()
	secretInformer.Informer()
	configMapInformer := kubeInformerFactory.Core().V1().ConfigMaps()
	configMapInformer.Informer()
	serviceAccountInformer := kubeInformerFactory.Core().V1().ServiceAccounts()
	serviceAccountInformer.Informer()
	deploymentInformer := kubeInformerFactory.Apps().V1beta2().Deployments()
	deploymentInformer.Informer()
	clusterInformer := clusterAPIInformerFactory.Cluster().V1alpha1().Clusters()
	clusterInformer.Informer()

	clusterInformer.Informer().AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	}, 30*time.Second)

	kubeInformerFactory.Start(wait.NeverStop)
	kubeInformerFactory.WaitForCacheSync(wait.NeverStop)

	clusterAPIInformerFactory.Start(wait.NeverStop)
	clusterAPIInformerFactory.WaitForCacheSync(wait.NeverStop)

	ctrl := controller.NewClusterController(
		queue,
		kubeClient,
		clusterClient,
		clusterInformer.Lister(),
		namespaceInformer.Lister(),
		serviceInformer.Lister(),
		secretInformer.Lister(),
		configMapInformer.Lister(),
		serviceAccountInformer.Lister(),
		deploymentInformer.Lister(),
	)

	ctx := context.Background()
	ctrl.Run(4, ctx.Done())
}
