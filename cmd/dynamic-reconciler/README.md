# Dynamic Reconciler

The dynamic reconciler is a flexible Kubernetes controller that can reconcile any Crossplane managed resource without having compile-time knowledge of the resource schemas. It connects to external provider servers via gRPC and coordinates the lifecycle management of external resources.

## Overview

The dynamic reconciler is designed to:

1. Connect to one or more provider gRPC servers
2. Dynamically register controllers for specified resource types
3. Reconcile resources using the standard Crossplane managed reconciler
4. Forward operations to the appropriate provider

This allows for a clean separation between Kubernetes controllers and external providers, without requiring compile-time type information.

## Usage

### Configuration

The dynamic reconciler can be configured either via a configuration file or command-line arguments:

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
  dynamic-reconciler [flags]

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

The dynamic reconciler uses the following components:

1. **Controller Runtime Manager**: Sets up the Kubernetes controllers and watch mechanisms
2. **Streaming gRPC Connector**: Establishes and maintains connections to provider servers
3. **Managed Reconciler**: Uses the standard Crossplane reconciliation logic
4. **Unstructured Resources**: Works with resources as unstructured data since schemas aren't known at compile time

### Workflow

1. On startup, the dynamic reconciler loads its configuration
2. It establishes gRPC connections to all configured providers
3. For each resource type, it registers a controller using controller-runtime
4. When a resource needs reconciliation, it:
   - Fetches the resource from Kubernetes
   - Connects to the appropriate provider server
   - Forwards operations (Observe, Create, Update, Delete) to the provider
   - Updates the resource status and/or finalizers in Kubernetes

## Building the Dynamic Reconciler

To build the dynamic reconciler as a container image:

```bash
# From the crossplane-runtime directory
docker build -t crossplane/dynamic-reconciler -f Dockerfile .
```

## Running in Kubernetes

The dynamic reconciler is designed to run in Kubernetes. Here's an example Deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dynamic-reconciler
spec:
  replicas: 1
  selector:
    matchLabels:
      app: dynamic-reconciler
  template:
    metadata:
      labels:
        app: dynamic-reconciler
    spec:
      containers:
      - name: reconciler
        image: crossplane/dynamic-reconciler:latest
        args:
        - --config=/etc/reconciler/config.json
        volumeMounts:
        - name: config
          mountPath: /etc/reconciler
      volumes:
      - name: config
        configMap:
          name: dynamic-reconciler-config
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: dynamic-reconciler-config
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