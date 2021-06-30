/*
Copyright 2015 The Kubernetes Authors.

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

// Package crdbootstrap contains all the logic for installing in-tree CRDs.
package crdbootstrap

import (
	"embed"
	"fmt"
	"io/fs"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/controller"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	informers "k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions/apiextensions/v1"
	listers "k8s.io/apiextensions-apiserver/pkg/client/listers/apiextensions/v1"
)

//go:embed crds/*.yaml
var inTreeCRDs embed.FS

type Controller struct {
	queue      workqueue.RateLimitingInterface
	crds       listers.CustomResourceDefinitionLister
	crdsSynced cache.InformerSynced
	crdStore   CRDStore
}

func NewController(informer informers.CustomResourceDefinitionInformer) (*Controller, error) {
	controller := &Controller{
		queue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "crd-bootstrap"),
		crds:  informer.Lister(),
	}

	store, err := NewInTreeCRDStoreFromFilesystem(inTreeCRDs)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("unable to initialize store: %v", err))
	}
	controller.crdStore = store

	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: controller.updateCRD,
		DeleteFunc: controller.deleteCRD,
	})

	return controller, nil
}

func (c *Controller) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.InfoS("starting crd bootstrap controller")
	defer klog.InfoS("stopping crd bootstrap controller")

	// install the CRDs present in tree
	if err := c.installInTree(); err != nil {
		klog.Fatalf("unable to install CRDs present in tree", err)
	}

	// wait for cache to be filled
	if !cache.WaitForNamedCacheSync("crd", stopCh, c.crdsSynced) {
		return
	}

	// run the worker in a loop
	go wait.Until(c.worker, time.Second, stopCh)

	// wait until we're told to stop
	<-stopCh
	klog.Infof("Shutting down crdbootstrap controller")
}

func (c *Controller) worker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	err := c.syncHandler(key.(string))
	c.handleErr(err, key)

	return true
}

func (c *Controller) syncHandler(key string) error {
	return nil
}

func (c *Controller) updateCRD(oldObj, newObj interface{}) {
	fmt.Println("update event registered")
}

func (c *Controller) deleteCRD(obj interface{}) {
	fmt.Println("delete event registered")
}

func (c *Controller) installInTree() error {
	for crd := range c.crdStore.List() {
		// TODO: install crd
		_ = crd
	}
	return nil
}

type CRDStore interface {
	ListKeys() ([]string, error)
	List() []*apiextensionsv1.CustomResourceDefinition
}

type InTreeCRDStore struct {
	CRDs []*apiextensionsv1.CustomResourceDefinition
}

func NewInTreeCRDStoreFromFilesystem(filesystem fs.FS) (CRDStore, error) {
	store := &InTreeCRDStore{}

	// TODO: walk through the filesystem and find all CRDs

	return store, nil
}

func (s InTreeCRDStore) ListKeys() ([]string, error) {
	keys := []string{}

	for _, crd := range s.CRDs {
		key, err := controller.KeyFunc(crd)

		if err != nil {
			utilruntime.HandleError(fmt.Errorf("couldn't get key for object %#v: %v", crd, err))
			return nil, nil
		}

		keys = append(keys, key)
	}

	return keys, nil
}

// TODO
func (s InTreeCRDStore) List() []*apiextensionsv1.CustomResourceDefinition {
	return nil, nil
}

// TODO
func ReadInTreeCRDs(filesystem fs.FS) ([]*apiextensionsv1.CustomResourceDefinition, error) {

	return []*apiextensionsv1.CustomResourceDefinition{}, nil
}
