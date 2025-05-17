# External Package

The external package provides implementations for client-server communication between Crossplane controllers and external providers using gRPC.

## Package Structure

The external package is organized into the following subpackages:

- `common`: Contains shared utilities and types used by both client and server implementations.
- `client`: Contains client-side implementations for connecting to and communicating with external providers.
- `server`: Contains server-side implementations for external providers.
- `remote`: **(DEPRECATED)** Legacy package for backward compatibility. This package will be removed in a future release. All code should use the above packages directly.

## Common Package

The common package provides shared functionality used by both clients and servers:

- Type conversion utilities between Kubernetes resources and protobuf structs
- Common types and interfaces for resource handlers
- Shared error handling and logging

## Client Package

The client package provides implementations for connecting to and communicating with external providers:

- `StreamingConnector`: Establishes gRPC connections to providers
- `StreamingClient`: Uses bidirectional streaming for more efficient communication
- Legacy client for backward compatibility

### Client Usage Example

```go
import (
    "context"
    "github.com/crossplane/crossplane-runtime/pkg/external/client"
)

// Create a streaming connector to the provider
connector := client.NewStreamingConnector(
    "provider-endpoint:9000", 
    tlsCredentials,
    client.WithClientLogger(log),
    client.WithResourceTypes(gvk),
)

// Connect to the provider for a specific resource
client, err := connector.Connect(ctx, myManagedResource)
if err != nil {
    return err
}

// Use the client to manage the external resource
observation, err := client.Observe(ctx, myManagedResource)
```

## Server Package

The server package provides implementations for the provider side:

- `ProviderServer`: Handles gRPC connections from controllers
- Resource type routing to appropriate handlers
- Session management for bidirectional streaming
- Legacy server for backward compatibility

### Server Usage Example

```go
import (
    "context"
    "github.com/crossplane/crossplane-runtime/pkg/external/server"
)

// Create a provider server
providerServer := server.NewProviderServer(
    scheme,
    server.WithServerLogger(log),
    server.WithTLSCredentials(serverCreds),
)

// Register handlers for resource types
providerServer.RegisterHandler(gvk, myExternalConnecter)

// Start the server
if err := providerServer.Serve(":9000"); err != nil {
    log.Fatal(err)
}
```

## Streaming vs Legacy API

The architecture supports two communication modes:

1. **Streaming API** (Recommended): Uses bidirectional streaming gRPC to maintain a persistent connection between the reconciler and provider components. This approach has several advantages:
   - Lower overhead for multiple operations on the same resource
   - Maintains connection state throughout a reconciliation cycle
   - Allows for more efficient resource usage

2. **Legacy API**: For backward compatibility, we also support the older non-streaming RPC interface. This allows existing providers to continue working while transitioning to the new architecture.

## Migrating from the `remote` package

To migrate from the deprecated `remote` package:

1. Replace imports from `pkg/external/remote` with either `pkg/external/client` or `pkg/external/server` as appropriate
2. Update any usage of `remote.Connector` to use `client.StreamingConnector`
3. Update any usage of `remote.ProviderServer` to use `server.ProviderServer`
4. Update any usage of the `remote.ConnectorManager` to use `client.ConnectorManager`

After ensuring no code depends on the `remote` package, you can run the `cleanup-remote.sh` script to remove it.
