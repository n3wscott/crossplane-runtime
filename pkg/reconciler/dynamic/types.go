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

// Package dynamic implements a dynamic reconciler that can work with any managed resource
// type without compile-time knowledge of the schema.
package dynamic

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ResourceType defines a resource type to be reconciled by a dynamic controller.
type ResourceType struct {
	// APIVersion is the API version of the resource.
	APIVersion string `json:"apiVersion"`

	// Kind is the kind of the resource.
	Kind string `json:"kind"`
}

// ProviderConfig defines the configuration for a dynamic provider.
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

// DynamicControllerConfig defines the configuration for the dynamic reconciler.
type DynamicControllerConfig struct {
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