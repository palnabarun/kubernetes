/*
Copyright 2016 The Kubernetes Authors.

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

package options

import (
	"fmt"
	"strings"
	"time"

	genericfeatures "k8s.io/apiserver/pkg/features"
	utilfeature "k8s.io/apiserver/pkg/util/feature"

	"github.com/spf13/pflag"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	authzconfig "k8s.io/apiserver/pkg/authorization/config"
	authzconfigloader "k8s.io/apiserver/pkg/authorization/config/load"
	authzconfigvalidation "k8s.io/apiserver/pkg/authorization/config/validation"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	versionedinformers "k8s.io/client-go/informers"
	"k8s.io/kubernetes/pkg/kubeapiserver/authorizer"
	authzmodes "k8s.io/kubernetes/pkg/kubeapiserver/authorizer/modes"
)

// BuiltInAuthorizationOptions contains all build-in authorization options for API Server
type BuiltInAuthorizationOptions struct {
	Modes                       []string
	PolicyFile                  string
	WebhookConfigFile           string
	WebhookVersion              string
	WebhookCacheAuthorizedTTL   time.Duration
	WebhookCacheUnauthorizedTTL time.Duration
	// WebhookRetryBackoff specifies the backoff parameters for the authorization webhook retry logic.
	// This allows us to configure the sleep time at each iteration and the maximum number of retries allowed
	// before we fail the webhook call in order to limit the fan out that ensues when the system is degraded.
	WebhookRetryBackoff *wait.Backoff

	AuthorizationConfigurationFile string
}

// NewBuiltInAuthorizationOptions create a BuiltInAuthorizationOptions with default value
func NewBuiltInAuthorizationOptions() *BuiltInAuthorizationOptions {
	return &BuiltInAuthorizationOptions{
		Modes:                       []string{authzmodes.ModeAlwaysAllow},
		WebhookVersion:              "v1beta1",
		WebhookCacheAuthorizedTTL:   5 * time.Minute,
		WebhookCacheUnauthorizedTTL: 30 * time.Second,
		WebhookRetryBackoff:         genericoptions.DefaultAuthWebhookRetryBackoff(),
	}
}

// Validate checks invalid config combination
func (o *BuiltInAuthorizationOptions) Validate() []error {
	if o == nil {
		return nil
	}
	var allErrors []error

	if !utilfeature.DefaultFeatureGate.Enabled(genericfeatures.StructuredAuthorizationConfig) && o.AuthorizationConfigurationFile != "" {
		allErrors = append(allErrors, fmt.Errorf("StructuredAuthorizationConfig is disabled but --authorization-config is set"))
	}

	if utilfeature.DefaultFeatureGate.Enabled(genericfeatures.StructuredAuthorizationConfig) {
		if o.AuthorizationConfigurationFile == "" {
			allErrors = append(allErrors, fmt.Errorf("StructuredAuthorizationConfig is enabled but --authorization-config is not set"))
		} else {
			config, err := authzconfigloader.LoadFromFile(o.AuthorizationConfigurationFile)
			if err != nil {
				allErrors = append(allErrors, fmt.Errorf("failed to load config from file %s: %s", o.AuthorizationConfigurationFile, err))
			}

			if errors := authzconfigvalidation.ValidateAuthorizationConfiguration(nil, config,
				sets.NewString(authzmodes.AuthorizationModeChoices...),
				sets.NewString(authzmodes.RepeatableAuthorizerTypes...),
			); len(errors) != 0 {
				allErrors = append(allErrors, errors.ToAggregate().Errors()...)
			}
		}
	} else {
		authzConfiguration := o.buildAuthorizationConfiguration()
		if errors := authzconfigvalidation.ValidateAuthorizationConfiguration(nil, authzConfiguration,
			sets.NewString(authzmodes.AuthorizationModeChoices...),
			sets.NewString(authzmodes.RepeatableAuthorizerTypes...),
		); len(errors) != 0 {
			allErrors = append(allErrors, errors.ToAggregate().Errors()...)
		}

	}

	modes := sets.NewString(o.Modes...)
	for _, mode := range o.Modes {
		if mode == authzmodes.ModeABAC && o.PolicyFile == "" {
			allErrors = append(allErrors, fmt.Errorf("authorization-mode ABAC's authorization policy file not passed"))
		}
	}

	if o.PolicyFile != "" && !modes.Has(authzmodes.ModeABAC) {
		allErrors = append(allErrors, fmt.Errorf("cannot specify --authorization-policy-file without mode ABAC"))
	}
	if o.WebhookConfigFile != "" && !modes.Has(authzmodes.ModeWebhook) {
		allErrors = append(allErrors, fmt.Errorf("cannot specify --authorization-webhook-config-file without mode Webhook"))
	}
	if o.WebhookRetryBackoff != nil && o.WebhookRetryBackoff.Steps <= 0 {
		allErrors = append(allErrors, fmt.Errorf("number of webhook retry attempts must be greater than 0, but is: %d", o.WebhookRetryBackoff.Steps))
	}

	return allErrors
}

// AddFlags returns flags of authorization for a API Server
func (o *BuiltInAuthorizationOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringSliceVar(&o.Modes, "authorization-mode", o.Modes, ""+
		"Ordered list of plug-ins to do authorization on secure port. Comma-delimited list of: "+
		strings.Join(authzmodes.AuthorizationModeChoices, ",")+".")

	fs.StringVar(&o.PolicyFile, "authorization-policy-file", o.PolicyFile, ""+
		"File with authorization policy in json line by line format, used with --authorization-mode=ABAC, on the secure port.")

	fs.StringVar(&o.WebhookConfigFile, "authorization-webhook-config-file", o.WebhookConfigFile, ""+
		"File with webhook configuration in kubeconfig format, used with --authorization-mode=Webhook. "+
		"The API server will query the remote service to determine access on the API server's secure port.")

	fs.StringVar(&o.WebhookVersion, "authorization-webhook-version", o.WebhookVersion, ""+
		"The API version of the authorization.k8s.io SubjectAccessReview to send to and expect from the webhook.")

	fs.DurationVar(&o.WebhookCacheAuthorizedTTL, "authorization-webhook-cache-authorized-ttl",
		o.WebhookCacheAuthorizedTTL,
		"The duration to cache 'authorized' responses from the webhook authorizer.")

	fs.DurationVar(&o.WebhookCacheUnauthorizedTTL,
		"authorization-webhook-cache-unauthorized-ttl", o.WebhookCacheUnauthorizedTTL,
		"The duration to cache 'unauthorized' responses from the webhook authorizer.")

	fs.StringVar(&o.AuthorizationConfigurationFile, "authorization-config", o.AuthorizationConfigurationFile, ""+
		"File with Authorization Configuration to configure the authorizer chain."+
		"Note: This feature is in Alpha since v1.28."+
		"The StructuredAuthorizationConfig feature needs to be set to true for enabling the functionality.")
}

// ToAuthorizationConfig convert BuiltInAuthorizationOptions to authorizer.Config
func (o *BuiltInAuthorizationOptions) ToAuthorizationConfig(versionedInformerFactory versionedinformers.SharedInformerFactory) (authorizer.Config, error) {
	// When the feature flag is enabled,
	//		the authorizer is built using the file provided through
	// 		--authorization-config and the legacy flags are disregarded, except --authorization-policy-file,
	//		which is required to define the ABAC Mode
	// Else,
	//		the AuthorizationConfiguration is built using the legacy flags.
	if utilfeature.DefaultFeatureGate.Enabled(genericfeatures.StructuredAuthorizationConfig) && o.AuthorizationConfigurationFile != "" {
		config, err := authzconfigloader.LoadFromFile(o.AuthorizationConfigurationFile)
		if err != nil {
			return authorizer.Config{}, fmt.Errorf("failed to load config from file: %s", err)
		}

		return authorizer.Config{
			PolicyFile:                 o.PolicyFile,
			VersionedInformerFactory:   versionedInformerFactory,
			WebhookRetryBackoff:        o.WebhookRetryBackoff,
			AuthorizationConfiguration: config,
		}, nil
	} else if utilfeature.DefaultFeatureGate.Enabled(genericfeatures.StructuredAuthorizationConfig) && o.AuthorizationConfigurationFile == "" {
		return authorizer.Config{}, fmt.Errorf("StructuredAuthorizationConfig is enabled but no configuration file is defined")
	} else {
		return authorizer.Config{
			PolicyFile:                 o.PolicyFile,
			VersionedInformerFactory:   versionedInformerFactory,
			WebhookRetryBackoff:        o.WebhookRetryBackoff,
			AuthorizationConfiguration: o.buildAuthorizationConfiguration(),
		}, nil
	}

}

// buildAuthorizationConfiguration converts existing flags to the AuthorizationConfiguration format
func (o *BuiltInAuthorizationOptions) buildAuthorizationConfiguration() *authzconfig.AuthorizationConfiguration {
	var authorizers []authzconfig.AuthorizerConfiguration
	for _, mode := range o.Modes {
		switch mode {
		case authzmodes.ModeWebhook:
			authorizers = append(authorizers, authzconfig.AuthorizerConfiguration{
				Type: authzconfig.TypeWebhook,
				Webhook: &authzconfig.WebhookConfiguration{
					Name:            authzconfig.DefaultWebhookName,
					AuthorizedTTL:   metav1.Duration{Duration: o.WebhookCacheAuthorizedTTL},
					UnauthorizedTTL: metav1.Duration{Duration: o.WebhookCacheUnauthorizedTTL},
					// Timeout and FailurePolicy are required for the new configuration.
					// Setting these two implicitly to preserve backward compatibility.
					Timeout:                    metav1.Duration{Duration: 30 * time.Second},
					FailurePolicy:              "NoOpinion",
					SubjectAccessReviewVersion: o.WebhookVersion,
					ConnectionInfo: authzconfig.WebhookConnectionInfo{
						Type:           "KubeConfigFile",
						KubeConfigFile: &o.WebhookConfigFile,
					},
				},
			})
		default:
			authorizers = append(authorizers, authzconfig.AuthorizerConfiguration{Type: authzconfig.AuthorizerType(mode)})
		}
	}
	return &authzconfig.AuthorizationConfiguration{Authorizers: authorizers}
}
