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

/*
Package dynamic implements a dynamic reconciler that can work with any managed resource
type without compile-time knowledge of the schema.

# Architecture

The package follows a layered architecture:

1. Configuration layer: Handles loading and validating configurations
2. Controller layer: Manages the overall lifecycle of the controllers
3. Provider layer: Handles connections to provider servers
4. Reconciliation layer: Uses Crossplane's managed reconciler for actual reconciliation

# Core Components

* DynamicControllerManager: Interface for managing dynamic controllers
* Provider: Manages connections to provider servers
* ResourceType: Represents a resource type that can be reconciled

# Usage

The package is designed to be used either through the provided CLI in cmd/dynamic-reconciler
or as a library in other Go programs:

	import "github.com/crossplane/crossplane-runtime/pkg/reconciler/dynamic"

	// Create a configuration
	config := dynamic.DynamicControllerConfig{...}

	// Build a controller
	builder := dynamic.NewDynamicControllerBuilder(config)
	controller, _ := builder.Build()

	// Setup and start
	controller.Setup(ctx)
	controller.Start(ctx)

# Extensions

The package is designed to be extended in various ways:

1. Custom provider implementations
2. Additional controller options
3. Custom resource type handling
*/
package dynamic