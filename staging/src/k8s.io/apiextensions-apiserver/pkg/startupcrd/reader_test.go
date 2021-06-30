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

package startupcrd

import (
	"testing"
)

const (
	validManifest = `
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: Operation
spec:
  group: bar.k8s.io
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                operand1:
                  type: string
                operand2:
                  type: string
				operator:
				  type: string
  scope: Namespaced
  names:
    plural: operations
    singular: operation
    kind: Operation
    shortNames:
    - op
`
	invalidManifest = `
foo: bar
`
)

func TestReadFromIOReader(t *testing.T) {
	//var tests = []struct {
	//	input string
	//	outputObjs []*unstructured.Unstructured
	//	outputErr error
	//}{
	//	{invalidManifest, nil, },
	//}
	//for _, test := range tests {
	//	t.Run()
	//}
}
