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

// This file was autogenerated by apiregister-gen. Do not edit it manually!

package sharedinformers

import (
	"github.com/kubernetes-incubator/apiserver-builder/pkg/controller"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-deploy/ext-apiserver/pkg/client/clientset_generated/clientset"
	"k8s.io/kube-deploy/ext-apiserver/pkg/client/informers_generated/externalversions"
	"time"
)

// SharedInformers wraps all informers used by controllers so that
// they are shared across controller implementations
type SharedInformers struct {
	controller.SharedInformersDefaults
	Factory externalversions.SharedInformerFactory
}

// newSharedInformers returns a set of started informers
func NewSharedInformers(config *rest.Config, shutdown <-chan struct{}) *SharedInformers {
	si := &SharedInformers{
		controller.SharedInformersDefaults{},
		externalversions.NewSharedInformerFactory(clientset.NewForConfigOrDie(config), 10*time.Minute),
	}
	if si.SetupKubernetesTypes() {
		si.InitKubernetesInformers(config)
	}
	si.Init()
	si.startInformers(shutdown)
	si.StartAdditionalInformers(shutdown)
	return si
}

// startInformers starts all of the informers
func (si *SharedInformers) startInformers(shutdown <-chan struct{}) {
	go si.Factory.Cluster().V1alpha1().Clusters().Informer().Run(shutdown)
	go si.Factory.Cluster().V1alpha1().Machines().Informer().Run(shutdown)
}

// ControllerInitArguments are arguments provided to the Init function for a new controller.
type ControllerInitArguments interface {
	// GetSharedInformers returns the SharedInformers that can be used to access
	// informers and listers for watching and indexing Kubernetes Resources
	GetSharedInformers() *SharedInformers

	// GetRestConfig returns the Config to create new client-go clients
	GetRestConfig() *rest.Config

	// Watch uses resourceInformer to watch a resource.  When create, update, or deletes
	// to the resource type are encountered, watch uses watchResourceToReconcileResourceKey
	// to lookup the key for the resource reconciled by the controller (maybe a different type
	// than the watched resource), and enqueue it to be reconciled.
	// watchName: name of the informer.  may appear in logs
	// resourceInformer: gotten from the SharedInformer.  controls which resource type is watched
	// getReconcileKey: takes an instance of the watched resource and returns
	//                                      a key for the reconciled resource type to enqueue.
	Watch(watchName string, resourceInformer cache.SharedIndexInformer,
		getReconcileKey func(interface{}) (string, error))
}

type ControllerInitArgumentsImpl struct {
	Si *SharedInformers
	Rc *rest.Config
	Rk func(key string) error
}

func (c ControllerInitArgumentsImpl) GetSharedInformers() *SharedInformers {
	return c.Si
}

func (c ControllerInitArgumentsImpl) GetRestConfig() *rest.Config {
	return c.Rc
}

// Watch uses resourceInformer to watch a resource.  When create, update, or deletes
// to the resource type are encountered, watch uses watchResourceToReconcileResourceKey
// to lookup the key for the resource reconciled by the controller (maybe a different type
// than the watched resource), and enqueue it to be reconciled.
// watchName: name of the informer.  may appear in logs
// resourceInformer: gotten from the SharedInformer.  controls which resource type is watched
// getReconcileKey: takes an instance of the watched resource and returns
//                                      a key for the reconciled resource type to enqueue.
func (c ControllerInitArgumentsImpl) Watch(
	watchName string, resourceInformer cache.SharedIndexInformer,
	getReconcileKey func(interface{}) (string, error)) {
	c.Si.Watch(watchName, resourceInformer, getReconcileKey, c.Rk)
}

type Controller interface{}

// LegacyControllerInit old controllers may implement this, and we keep
// it for backwards compatibility.
type LegacyControllerInit interface {
	Init(config *rest.Config, si *SharedInformers, r func(key string) error)
}

// ControllerInit new controllers should implement this.  It is more flexible in
// allowing additional options to be passed in
type ControllerInit interface {
	Init(args ControllerInitArguments)
}
