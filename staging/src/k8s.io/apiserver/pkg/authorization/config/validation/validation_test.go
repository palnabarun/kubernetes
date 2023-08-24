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
	"os"
	"time"

	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
	api "k8s.io/apiserver/pkg/authorization/config"
)

type (
	test struct {
		configuration   api.AuthorizationConfiguration
		expectedErrList field.ErrorList
		knownTypes      sets.Set[string]
		repeatableTypes sets.Set[string]
	}
)

func TestValidateAuthorizationConfiguration(t *testing.T) {
	badKubeConfigFile := "../some/relative/path/kubeconfig"

	tempKubeConfigFile, err := os.CreateTemp("/tmp", "kubeconfig")
	if err != nil {
		t.Fatalf("failed to set up temp file: %v", err)
	}
	tempKubeConfigFilePath := tempKubeConfigFile.Name()
	defer os.Remove(tempKubeConfigFilePath)

	tests := []test{
		// atleast one authorizer should be defined
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{},
			},
			expectedErrList: field.ErrorList{field.Required(field.NewPath("authorizers"), "at least one authorization mode must be defined")},
			knownTypes:      sets.New[string](),
			repeatableTypes: sets.New[string](),
		},
		// type is required if an authorizer is defined
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{},
				},
			},
			expectedErrList: field.ErrorList{field.Required(field.NewPath("type"), "")},
			knownTypes:      sets.New[string]("Webhook"),
			repeatableTypes: sets.New[string]("Webhook"),
		},
		// bare minimum configuration with Webhook
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Webhook",
						Webhook: &api.WebhookConfiguration{
							Name:                       "default",
							Timeout:                    metav1.Duration{Duration: 5 * time.Second},
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
			knownTypes:      sets.New[string]("Webhook"),
			repeatableTypes: sets.New[string]("Webhook"),
		},
		// bare minimum configuration with multiple webhooks
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Webhook",
						Webhook: &api.WebhookConfiguration{
							Name:                       "default",
							Timeout:                    metav1.Duration{Duration: 5 * time.Second},
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
							Timeout:                    metav1.Duration{Duration: 5 * time.Second},
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
			knownTypes:      sets.New[string]("Webhook"),
			repeatableTypes: sets.New[string]("Webhook"),
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
			knownTypes:      sets.New[string]("Webhook"),
			repeatableTypes: sets.New[string]("Webhook"),
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
			knownTypes:      sets.New[string]("Webhook"),
			repeatableTypes: sets.New[string]("Webhook"),
		},
		// when type=Webhook, webhook needs to be defined
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Webhook",
					},
				},
			},
			expectedErrList: field.ErrorList{field.Required(field.NewPath("webhook"), "required when type=Webhook")},
			knownTypes:      sets.New[string]("Webhook"),
			repeatableTypes: sets.New[string]("Webhook"),
		},
		// when type!=Webhook, webhooks needs to be nil
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type:    "Foo",
						Webhook: &api.WebhookConfiguration{},
					},
				},
			},
			expectedErrList: field.ErrorList{field.Invalid(field.NewPath("webhook"), "non-null", "may only be specified when type=Webhook")},
			knownTypes:      sets.New[string]("Foo"),
			repeatableTypes: sets.New[string]("Webhook"),
		},
		// webhook name should be of non-zero length
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Webhook",
						Webhook: &api.WebhookConfiguration{
							Name:                       "",
							Timeout:                    metav1.Duration{Duration: 5 * time.Second},
							FailurePolicy:              "NoOpinion",
							SubjectAccessReviewVersion: "v1",
							ConnectionInfo: api.WebhookConnectionInfo{
								Type: "InClusterConfig",
							},
						},
					},
				},
			},
			expectedErrList: field.ErrorList{field.Required(field.NewPath("name"), "")},
			knownTypes:      sets.New[string]("Webhook"),
			repeatableTypes: sets.New[string]("Webhook"),
		},
		// webhook names should be unique
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Webhook",
						Webhook: &api.WebhookConfiguration{
							Name:                       "name-1",
							Timeout:                    metav1.Duration{Duration: 5 * time.Second},
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
							Name:                       "name-1",
							Timeout:                    metav1.Duration{Duration: 5 * time.Second},
							FailurePolicy:              "NoOpinion",
							SubjectAccessReviewVersion: "v1",
							ConnectionInfo: api.WebhookConnectionInfo{
								Type: "InClusterConfig",
							},
						},
					},
				},
			},
			expectedErrList: field.ErrorList{field.Duplicate(field.NewPath("name"), "name-1")},
			knownTypes:      sets.New[string]("Webhook"),
			repeatableTypes: sets.New[string]("Webhook"),
		},
		// webhook names should conform to RFC1035
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Webhook",
						Webhook: &api.WebhookConfiguration{
							Name:                       "1-lorem-ipsum-doler-es-mit",
							Timeout:                    metav1.Duration{Duration: 5 * time.Second},
							FailurePolicy:              "NoOpinion",
							SubjectAccessReviewVersion: "v1",
							ConnectionInfo: api.WebhookConnectionInfo{
								Type: "InClusterConfig",
							},
						},
					},
				},
			},
			expectedErrList: field.ErrorList{field.Invalid(field.NewPath("name"), "1-lorem-ipsum-doler-es-mit", "webhook name is invalid")},
			knownTypes:      sets.New[string]("Webhook"),
			repeatableTypes: sets.New[string]("Webhook"),
		},
		// timeout should be specified
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Webhook",
						Webhook: &api.WebhookConfiguration{
							Name:                       "default",
							FailurePolicy:              "NoOpinion",
							SubjectAccessReviewVersion: "v1",
							ConnectionInfo: api.WebhookConnectionInfo{
								Type: "InClusterConfig",
							},
						},
					},
				},
			},
			expectedErrList: field.ErrorList{field.Required(field.NewPath("timeout"), "")},
			knownTypes:      sets.New[string]("Webhook"),
			repeatableTypes: sets.New[string]("Webhook"),
		},
		// timeout shouldn't be zero
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Webhook",
						Webhook: &api.WebhookConfiguration{
							Name:                       "default",
							FailurePolicy:              "NoOpinion",
							Timeout:                    metav1.Duration{Duration: 0 * time.Second},
							SubjectAccessReviewVersion: "v1",
							ConnectionInfo: api.WebhookConnectionInfo{
								Type: "InClusterConfig",
							},
						},
					},
				},
			},
			expectedErrList: field.ErrorList{field.Required(field.NewPath("timeout"), "")},
			knownTypes:      sets.New[string]("Webhook"),
			repeatableTypes: sets.New[string]("Webhook"),
		},
		// timeout shouldn't be greater than 30seconds
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Webhook",
						Webhook: &api.WebhookConfiguration{
							Name:                       "default",
							FailurePolicy:              "NoOpinion",
							Timeout:                    metav1.Duration{Duration: 60 * time.Second},
							SubjectAccessReviewVersion: "v1",
							ConnectionInfo: api.WebhookConnectionInfo{
								Type: "InClusterConfig",
							},
						},
					},
				},
			},
			expectedErrList: field.ErrorList{field.Invalid(field.NewPath("timeout"), time.Duration(60*time.Second).String(), "must be <= 30s")},
			knownTypes:      sets.New[string]("Webhook"),
			repeatableTypes: sets.New[string]("Webhook"),
		},
		// SAR should defined
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Webhook",
						Webhook: &api.WebhookConfiguration{
							Name:          "default",
							Timeout:       metav1.Duration{Duration: 5 * time.Second},
							FailurePolicy: "NoOpinion",
							ConnectionInfo: api.WebhookConnectionInfo{
								Type: "InClusterConfig",
							},
						},
					},
				},
			},
			expectedErrList: field.ErrorList{field.Required(field.NewPath("subjectAccessReviewVersion"), "")},
			knownTypes:      sets.New[string]("Webhook"),
			repeatableTypes: sets.New[string]("Webhook"),
		},
		// SAR should be one of v1 and v1beta1
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Webhook",
						Webhook: &api.WebhookConfiguration{
							Name:                       "default",
							Timeout:                    metav1.Duration{Duration: 5 * time.Second},
							FailurePolicy:              "NoOpinion",
							SubjectAccessReviewVersion: "v2beta1",
							ConnectionInfo: api.WebhookConnectionInfo{
								Type: "InClusterConfig",
							},
						},
					},
				},
			},
			expectedErrList: field.ErrorList{field.NotSupported(field.NewPath("subjectAccessReviewVersion"), "v2beta1", []string{"v1", "v1beta1"})},
			knownTypes:      sets.New[string]("Webhook"),
			repeatableTypes: sets.New[string]("Webhook"),
		},
		// failurePolicy should be defined
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Webhook",
						Webhook: &api.WebhookConfiguration{
							Name:                       "default",
							Timeout:                    metav1.Duration{Duration: 5 * time.Second},
							SubjectAccessReviewVersion: "v1",
							ConnectionInfo: api.WebhookConnectionInfo{
								Type: "InClusterConfig",
							},
						},
					},
				},
			},
			expectedErrList: field.ErrorList{field.Required(field.NewPath("failurePolicy"), "")},
			knownTypes:      sets.New[string]("Webhook"),
			repeatableTypes: sets.New[string]("Webhook"),
		},
		// failurePolicy should be one of "NoOpinion" or "Deny"
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Webhook",
						Webhook: &api.WebhookConfiguration{
							Name:                       "default",
							Timeout:                    metav1.Duration{Duration: 5 * time.Second},
							FailurePolicy:              "AlwaysAllow",
							SubjectAccessReviewVersion: "v1",
							ConnectionInfo: api.WebhookConnectionInfo{
								Type: "InClusterConfig",
							},
						},
					},
				},
			},
			expectedErrList: field.ErrorList{field.NotSupported(field.NewPath("failurePolicy"), "AlwaysAllow", []string{"NoOpinion", "Deny"})},
			knownTypes:      sets.New[string]("Webhook"),
			repeatableTypes: sets.New[string]("Webhook"),
		},
		// connectionInfo should be defined
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Webhook",
						Webhook: &api.WebhookConfiguration{
							Name:                       "default",
							Timeout:                    metav1.Duration{Duration: 5 * time.Second},
							FailurePolicy:              "NoOpinion",
							SubjectAccessReviewVersion: "v1",
						},
					},
				},
			},
			expectedErrList: field.ErrorList{field.Required(field.NewPath("connectionInfo"), "")},
			knownTypes:      sets.New[string]("Webhook"),
			repeatableTypes: sets.New[string]("Webhook"),
		},
		// connectionInfo should be one of InClusterConfig or KubeConfigFile
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Webhook",
						Webhook: &api.WebhookConfiguration{
							Name:                       "default",
							Timeout:                    metav1.Duration{Duration: 5 * time.Second},
							FailurePolicy:              "NoOpinion",
							SubjectAccessReviewVersion: "v1",
							ConnectionInfo: api.WebhookConnectionInfo{
								Type: "ExternalClusterConfig",
							},
						},
					},
				},
			},
			expectedErrList: field.ErrorList{
				field.NotSupported(field.NewPath("connectionInfo"), api.WebhookConnectionInfo{Type: "ExternalClusterConfig"}, []string{"InClusterConfig", "KubeConfigFile"}),
			},
			knownTypes:      sets.New[string]("Webhook"),
			repeatableTypes: sets.New[string]("Webhook"),
		},
		// if connectionInfo=InClusterConfig, then kubeConfigFile should be nil
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Webhook",
						Webhook: &api.WebhookConfiguration{
							Name:                       "default",
							Timeout:                    metav1.Duration{Duration: 5 * time.Second},
							FailurePolicy:              "NoOpinion",
							SubjectAccessReviewVersion: "v1",
							ConnectionInfo: api.WebhookConnectionInfo{
								Type:           "InClusterConfig",
								KubeConfigFile: new(string),
							},
						},
					},
				},
			},
			expectedErrList: field.ErrorList{
				field.Invalid(field.NewPath("connectionInfo", "kubeConfigFile"), "", "can only be set when type=KubeConfigFile"),
			},
			knownTypes:      sets.New[string]("Webhook"),
			repeatableTypes: sets.New[string]("Webhook"),
		},
		// if connectionInfo=KubeConfigFile, then KubeConfigFile should be defined, must be an absolute path, should exist, shouldn't be a symlink
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Webhook",
						Webhook: &api.WebhookConfiguration{
							Name:                       "default",
							Timeout:                    metav1.Duration{Duration: 5 * time.Second},
							FailurePolicy:              "NoOpinion",
							SubjectAccessReviewVersion: "v1",
							ConnectionInfo: api.WebhookConnectionInfo{
								Type: "KubeConfigFile",
							},
						},
					},
				},
			},
			expectedErrList: field.ErrorList{field.Required(field.NewPath("kubeConfigFile"), "")},
			knownTypes:      sets.New[string]("Webhook"),
			repeatableTypes: sets.New[string]("Webhook"),
		},
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Webhook",
						Webhook: &api.WebhookConfiguration{
							Name:                       "default",
							Timeout:                    metav1.Duration{Duration: 5 * time.Second},
							FailurePolicy:              "NoOpinion",
							SubjectAccessReviewVersion: "v1",
							ConnectionInfo: api.WebhookConnectionInfo{
								Type:           "KubeConfigFile",
								KubeConfigFile: &badKubeConfigFile,
							},
						},
					},
				},
			},
			expectedErrList: field.ErrorList{field.Invalid(field.NewPath("kubeConfigFile"), badKubeConfigFile, "must be an absolute path")},
			knownTypes:      sets.New[string]("Webhook"),
			repeatableTypes: sets.New[string]("Webhook"),
		},
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Webhook",
						Webhook: &api.WebhookConfiguration{
							Name:                       "default",
							Timeout:                    metav1.Duration{Duration: 5 * time.Second},
							FailurePolicy:              "NoOpinion",
							SubjectAccessReviewVersion: "v1",
							ConnectionInfo: api.WebhookConnectionInfo{
								Type:           "KubeConfigFile",
								KubeConfigFile: &tempKubeConfigFilePath,
							},
						},
					},
				},
			},
			expectedErrList: field.ErrorList{},
			knownTypes:      sets.New[string]("Webhook"),
			repeatableTypes: sets.New[string]("Webhook"),
		},
		// if matchConditions are defined, the expression should be non-empty
		{
			configuration: api.AuthorizationConfiguration{
				Authorizers: []api.AuthorizerConfiguration{
					{
						Type: "Webhook",
						Webhook: &api.WebhookConfiguration{
							Name:                       "default",
							Timeout:                    metav1.Duration{Duration: 5 * time.Second},
							FailurePolicy:              "NoOpinion",
							SubjectAccessReviewVersion: "v1",
							ConnectionInfo: api.WebhookConnectionInfo{
								Type: "InClusterConfig",
							},
							MatchConditions: []api.WebhookMatchCondition{
								{
									Expression: "",
								},
							},
						},
					},
				},
			},
			expectedErrList: field.ErrorList{field.Required(field.NewPath("expression"), "")},
			knownTypes:      sets.New[string]("Webhook"),
			repeatableTypes: sets.New[string]("Webhook"),
		},

		// TODO: When the CEL expression validator is implemented, add a few test cases to typecheck the expression
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
