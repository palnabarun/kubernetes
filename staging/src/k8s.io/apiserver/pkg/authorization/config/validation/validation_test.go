/*
Copyright 2023 The Kubernetes Authors.

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

package validation

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"time"

	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
	api "k8s.io/apiserver/pkg/authorization/config"
)

type (
	test struct {
		configuration   api.AuthorizationConfiguration
		expectedErrList field.ErrorList
		knownTypes      sets.String
		repeatableTypes sets.String
	}
)

func TestValidateAuthorizationConfiguration(t *testing.T) {
	tests := []test{
		// type is required if an authorizer is defined
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{},
				},
			},
			expectedErrList: field.ErrorList{field.Required(field.NewPath("type"), "")},
			knownTypes:      sets.NewString(string("Webhook")),
			repeatableTypes: sets.NewString(string("Webhook")),
		},
		// bare minimum configuration with Webhook
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Webhook",
						Webhook: &api.WebhookConfiguration{
							Name:                       "default",
							Timeout:                    metav1.Duration{5 * time.Second},
							FailurePolicy:              "NoOpinion",
							SubjectAccessReviewVersion: "v1",
							ConnectionInfo: api.WebhookConnectionInfo{
								Type: "InClusterConfig",
							},
						},
					},
				},
			},
			expectedErrList: field.ErrorList{},
			knownTypes:      sets.NewString(string("Webhook")),
			repeatableTypes: sets.NewString(string("Webhook")),
		},
		// bare minimum configuration with multiple webhooks
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Webhook",
						Webhook: &api.WebhookConfiguration{
							Name:                       "default",
							Timeout:                    metav1.Duration{5 * time.Second},
							FailurePolicy:              "NoOpinion",
							SubjectAccessReviewVersion: "v1",
							ConnectionInfo: api.WebhookConnectionInfo{
								Type: "InClusterConfig",
							},
						},
					},
					{
						Type: "Webhook",
						Webhook: &api.WebhookConfiguration{
							Name:                       "second-webhook",
							Timeout:                    metav1.Duration{5 * time.Second},
							FailurePolicy:              "NoOpinion",
							SubjectAccessReviewVersion: "v1",
							ConnectionInfo: api.WebhookConnectionInfo{
								Type: "InClusterConfig",
							},
						},
					},
				},
			},
			expectedErrList: field.ErrorList{},
			knownTypes:      sets.NewString(string("Webhook")),
			repeatableTypes: sets.NewString(string("Webhook")),
		},
		// configuration with unknown types
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Foo",
					},
				},
			},
			expectedErrList: field.ErrorList{field.NotSupported(field.NewPath("type"), "Foo", []string{"..."})},
			knownTypes:      sets.NewString(string("Webhook")),
			repeatableTypes: sets.NewString(string("Webhook")),
		},
		// configuration with not repeatable types
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Foo",
					},
					{
						Type: "Foo",
					},
				},
			},
			expectedErrList: field.ErrorList{field.Duplicate(field.NewPath("type"), "Foo")},
			knownTypes:      sets.NewString([]string{string("Foo"), string("Webhook")}...),
			repeatableTypes: sets.NewString(string("Webhook")),
		},
	}

	for _, test := range tests {
		errList := ValidateAuthorizationConfiguration(nil, &test.configuration, test.knownTypes, test.repeatableTypes)
		if len(errList) != len(test.expectedErrList) {
			t.Errorf("expected %d errs, got %d, errors %v", len(test.expectedErrList), len(errList), errList)
		}

		for i, expected := range test.expectedErrList {
			if expected.Type.String() != errList[i].Type.String() {
				t.Errorf("expected err type %s, got %s",
					expected.Type.String(),
					errList[i].Type.String())
			}
			if expected.BadValue != errList[i].BadValue {
				t.Errorf("expected bad value '%s', got '%s'",
					expected.BadValue,
					errList[i].BadValue)
			}
		}
	}
}
