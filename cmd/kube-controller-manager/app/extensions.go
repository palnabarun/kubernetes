/*
Copyright 2021 The Kubernetes Authors.

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

// Package app implements a server that runs a set of active
// components.  This includes replication controllers, service endpoints and
// nodes.
//
package app

import (
	"fmt"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/controller/crdbootstrap"
	"net/http"
	"time"

	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	externalinformers "k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions"
)

func startCRDBootstrapController(ctx ControllerContext) (http.Handler, bool, error) {
	// get clientset for accessing ApiExtensions
	crdClient, err := clientset.NewForConfig(ctx.ClientBuilder.ConfigOrDie("crd-bootstrap"))
	if err != nil {
		// it's really bad that this is leaking here, but until we can fix the test (which I'm pretty sure isn't even testing what it wants to test),
		// we need to be able to move forward
		return nil, false, fmt.Errorf("failed to create clientset: %v", err)
	}
	// get a SharedInformerFactory for ApiExtensions
	informerFactory := externalinformers.NewSharedInformerFactory(crdClient, 1*time.Minute)

	controller, err := crdbootstrap.NewController(
		informerFactory.Apiextensions().V1().CustomResourceDefinitions(),
	)
	if err != nil {
		klog.Errorf("Failed to start service controller: %v", err)
		return nil, false, nil
	}
	go controller.Run(ctx.Stop)
	return nil, true, nil
}
