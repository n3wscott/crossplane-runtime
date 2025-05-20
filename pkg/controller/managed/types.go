/*
Copyright 2025 The Crossplane Authors.
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

package managed

import (
	"encoding/json"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"os"
)

// ResourceType defines a resource type to be reconciled by a managed controller.
type ResourceType struct {
	// APIVersion is the API version of the resource.
	APIVersion string `json:"apiVersion"`

	// Kind is the kind of the resource.
	Kind string `json:"kind"`
}

// ProviderConfig defines the configuration for a managed provider.
type ProviderConfig struct {
	// Name is a unique name for this provider.
	Name string `json:"name"`

	// Endpoint is the gRPC endpoint of the provider.
	Endpoint string `json:"endpoint"`

	// UseSSL indicates whether to use SSL for the connection.
	UseSSL bool `json:"useSSL,omitempty"`

	// ResourceTypes is a list of resource types this provider supports.
	ResourceTypes []ResourceType `json:"resourceTypes"`
}

// ControllerConfig defines the configuration for the managed reconciler.
type ControllerConfig struct {
	// Providers is a list of provider configurations.
	Providers []ProviderConfig `json:"providers"`
}

// ResourceTypeToGVK converts a ResourceType to a GroupVersionKind.
func ResourceTypeToGVK(rt ResourceType) (schema.GroupVersionKind, error) {
	gv, err := schema.ParseGroupVersion(rt.APIVersion)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	return gv.WithKind(rt.Kind), nil
}

// ConfigOption configures how config is loaded.
type ConfigOption func(*ConfigOptions)

// ConfigOptions defines options for loading configurations.
type ConfigOptions struct {
	// Default provider endpoint when none is specified.
	DefaultEndpoint string

	// Default provider name when none is specified.
	DefaultName string
}

// WithDefaultEndpoint sets the default provider endpoint.
func WithDefaultEndpoint(endpoint string) ConfigOption {
	return func(o *ConfigOptions) {
		o.DefaultEndpoint = endpoint
	}
}

// WithDefaultName sets the default provider name.
func WithDefaultName(name string) ConfigOption {
	return func(o *ConfigOptions) {
		o.DefaultName = name
	}
}

// DefaultConfigOptions returns the default ConfigOptions.
func DefaultConfigOptions() *ConfigOptions {
	return &ConfigOptions{
		DefaultEndpoint: "localhost:50051",
		DefaultName:     "default",
	}
}

// LoadConfigFromFile loads a ControllerConfig from a file.
func LoadConfigFromFile(path string, opts ...ConfigOption) (ControllerConfig, error) {
	// Apply options
	options := DefaultConfigOptions()
	for _, opt := range opts {
		opt(options)
	}

	// Read and parse the config file
	data, err := os.ReadFile(path)
	if err != nil {
		return ControllerConfig{}, errors.Wrap(err, "unable to read config file")
	}

	var config ControllerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return ControllerConfig{}, errors.Wrap(err, "unable to parse config file")
	}

	return config, nil
}

// CreateConfigFromEndpoint creates a ControllerConfig from a single endpoint.
func CreateConfigFromEndpoint(endpoint string, opts ...ConfigOption) ControllerConfig {
	// Apply options
	options := DefaultConfigOptions()
	for _, opt := range opts {
		opt(options)
	}

	// If no endpoint is provided, use the default
	if endpoint == "" {
		endpoint = options.DefaultEndpoint
	}
	
	return ControllerConfig{
		Providers: []ProviderConfig{
			{
				Name:     options.DefaultName,
				Endpoint: endpoint,
			},
		},
	}
}

// ValidateConfig checks if a ControllerConfig is valid.
func ValidateConfig(config ControllerConfig) error {
	if len(config.Providers) == 0 {
		return errors.New("no providers specified in configuration")
	}

	for i, provider := range config.Providers {
		if provider.Name == "" {
			return errors.Errorf("provider at index %d has no name", i)
		}
		if provider.Endpoint == "" {
			return errors.Errorf("provider %s has no endpoint", provider.Name)
		}
	}

	return nil
}
