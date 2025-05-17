# Dynamic Reconciler Package

The dynamic reconciler package provides components for building dynamic controllers that can reconcile Crossplane managed resources without compile-time knowledge of the resource schemas.

## Overview

This package implements a flexible system for reconciling managed resources where the type information is only known at runtime. It uses gRPC to communicate with provider servers that handle the external resource management.

## Components

### Types

The `types.go` file defines the core data structures:

- `ResourceType`: Represents a resource type that can be reconciled
- `ProviderConfig`: Configuration for a provider including endpoint and supported resource types
- `DynamicControllerConfig`: Top-level configuration for dynamic controllers

### Configuration

The `config.go` file provides utilities for loading and validating configurations:

- `LoadConfigFromFile`: Load configuration from a JSON file
- `CreateConfigFromEndpoint`: Create configuration from a single endpoint
- `ValidateConfig`: Verify that a configuration is valid

### Provider

The `provider.go` file defines the `Provider` type that manages connections to provider servers:

- `NewProvider`: Create a new provider with given configuration
- `Setup`: Register controllers for all resource types
- `setupResourceController`: Setup a controller for a specific resource type

### Controller

The `controller.go` file defines the main controller management:

- `DynamicControllerManager`: Interface for controller management
- `DynamicControllerBuilder`: Builder pattern for creating controllers
- `DynamicController`: Implementation of the manager interface

## Usage

To use this package in your own project:

```go
import "github.com/crossplane/crossplane-runtime/pkg/reconciler/dynamic"

// Create a configuration
config := dynamic.DynamicControllerConfig{
    Providers: []dynamic.ProviderConfig{
        {
            Name: "example",
            Endpoint: "provider:50051",
            ResourceTypes: []dynamic.ResourceType{
                {
                    APIVersion: "example.org/v1",
                    Kind: "ExampleResource",
                },
            },
        },
    },
}

// Create a controller builder
builder := dynamic.NewDynamicControllerBuilder(config,
    dynamic.WithLogger(logger),
    dynamic.WithMetricsAddress(":8080"),
    dynamic.WithPollInterval(1*time.Minute),
)

// Build and run the controller
controller, err := builder.Build()
if err != nil {
    // Handle error
}

if err := controller.Setup(ctx); err != nil {
    // Handle error
}

if err := controller.Start(ctx); err != nil {
    // Handle error
}
```

## Extending

You can extend this package in various ways:

1. Add new configuration options
2. Implement custom provider types
3. Add support for new authentication methods
4. Extend the controller manager with additional functionality