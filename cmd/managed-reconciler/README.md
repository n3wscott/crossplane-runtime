# Managed Reconciler

The managed reconciler is a flexible Kubernetes controller that can reconcile any Crossplane managed resource without having compile-time knowledge of the resource schemas. It connects to external provider servers via gRPC and coordinates the lifecycle management of external resources.

## Overview

The managed reconciler is designed to:

1. Connect to one or more provider gRPC servers
2. Dynamically register controllers for specified resource types
3. Reconcile resources using the standard Crossplane managed reconciler
4. Forward operations to the appropriate provider

This allows for a clean separation between Kubernetes controllers and external providers, without requiring compile-time type information.

## Package Structure

The managed reconciler is implemented in the `pkg/controller/managed` package with the following components:

- `types.go`: Type definitions for configuration and resources, plus configuration handling and validation
- `provider.go`: Provider connection management and controller setup
- `controller.go`: Overall controller management and orchestration
- `doc.go`: Package documentation and usage examples

This modular design makes it easy to reuse the managed reconciler components in other contexts.

## Usage

### Configuration

The managed reconciler can be configured either via a configuration file or command-line arguments:

```json
{
  "providers": [
    {
      "name": "example-provider",
      "endpoint": "provider-example:50051",
      "useSSL": false,
      "resourceTypes": [
        {
          "apiVersion": "example.crossplane.io/v1alpha1",
          "kind": "ExampleResource"
        },
        {
          "apiVersion": "example.crossplane.io/v1alpha1",
          "kind": "AnotherResource"
        }
      ]
    }
  ]
}
```

### Command-Line Options

```
Usage:
  managed-reconciler [flags]

Flags:
      --config string                      Path to the configuration file
      --provider-endpoint string           gRPC endpoint for the provider (overrides config file)
      --leader-election                    Use leader election for the controller (default true)
      --restart-on-provider-disconnect     Restart the reconciler if the provider connection is lost (default true)
      --max-reconcile-rate int             The maximum number of concurrent reconciliations per controller (default 10)
      --poll-interval duration             How often a managed resource should be polled when in a steady state (default 1m0s)
      --metrics-bind-address string        The address the metric endpoint binds to (default ":8080")
      --health-probe-bind-address string   The address the probe endpoint binds to (default ":8081")
      --cert-dir string                    The directory containing TLS certificates
```

## Architecture

The managed reconciler uses the following components:

1. **Controller Runtime Manager**: Sets up the Kubernetes controllers and watch mechanisms
2. **Engine**: Manages dynamic controllers and watches efficiently
3. **Streaming gRPC Connector**: Establishes and maintains connections to provider servers
4. **Managed Reconciler**: Uses the standard Crossplane reconciliation logic
5. **Unstructured Resources**: Works with resources as unstructured data since schemas aren't known at compile time

### Workflow

1. On startup, the managed reconciler loads its configuration
2. It establishes gRPC connections to all configured providers
3. It creates an engine for dynamic controller management
4. For each resource type, it registers a controller using the engine
5. When a resource needs reconciliation, it:
   - Fetches the resource from Kubernetes
   - Connects to the appropriate provider server
   - Forwards operations (Observe, Create, Update, Delete) to the provider
   - Updates the resource status and/or finalizers in Kubernetes

## Building the Managed Reconciler

To build the managed reconciler:

```bash
# From the crossplane-runtime directory
earthly +build
```

## Running in Kubernetes

The managed reconciler is designed to run in Kubernetes. Here's an example Deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: managed-reconciler
spec:
  replicas: 1
  selector:
    matchLabels:
      app: managed-reconciler
  template:
    metadata:
      labels:
        app: managed-reconciler
    spec:
      containers:
      - name: reconciler
        image: crossplane/managed-reconciler:latest
        args:
        - --config=/etc/reconciler/config.json
        volumeMounts:
        - name: config
          mountPath: /etc/reconciler
      volumes:
      - name: config
        configMap:
          name: managed-reconciler-config
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: managed-reconciler-config
data:
  config.json: |
    {
      "providers": [
        {
          "name": "example-provider",
          "endpoint": "provider-example:50051",
          "resourceTypes": [
            {
              "apiVersion": "example.crossplane.io/v1alpha1",
              "kind": "ExampleResource"
            }
          ]
        }
      ]
    }
```

## Programmatic Usage

The managed reconciler can also be used as a library in other Go programs:

```go
import "github.com/crossplane/crossplane-runtime/pkg/controller/managed"

// Create a configuration
config := managed.ControllerConfig{
    Providers: []managed.ProviderConfig{
        {
            Name:     "example-provider",
            Endpoint: "localhost:50051",
            ResourceTypes: []managed.ResourceType{
                {
                    APIVersion: "example.crossplane.io/v1alpha1",
                    Kind:       "ExampleResource",
                },
            },
        },
    },
}

// Create and start the controller
controller := managed.NewController(config, 
    managed.WithLogger(logger),
    managed.WithLeaderElection(true),
)

// Start the controller
if err := controller.Start(ctx); err != nil {
    panic(err)
}
```