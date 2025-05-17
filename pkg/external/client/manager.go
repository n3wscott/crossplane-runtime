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

package client

import (
	"context"
	"sync"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

// ConnectorManager manages gRPC connectors for different providers.
type ConnectorManager struct {
	// mu protects the connectors map.
	mu sync.RWMutex

	// connectors maps provider endpoints to StreamingConnector instances.
	connectors map[string]*StreamingConnector

	// log is the logger to use.
	log logging.Logger
}

// NewConnectorManager creates a new ConnectorManager.
func NewConnectorManager(log logging.Logger) *ConnectorManager {
	return &ConnectorManager{
		connectors: make(map[string]*StreamingConnector),
		log:        log,
	}
}

// GetOrCreateConnector gets or creates a StreamingConnector for the given endpoint.
// If a connector doesn't exist, it creates a new one.
func (m *ConnectorManager) GetOrCreateConnector(endpoint string, creds credentials.TransportCredentials, gvk schema.GroupVersionKind, opts ...StreamingConnectorOption) (*StreamingConnector, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If a connector already exists, return it
	if connector, ok := m.connectors[endpoint]; ok {
		// Add the GVK to the connector's supported types
		connector.gvkMap[gvk] = struct{}{}
		return connector, nil
	}

	// Create a new connector
	connectorOpts := append([]StreamingConnectorOption{
		WithResourceTypes(gvk),
		WithClientLogger(m.log.WithValues("endpoint", endpoint)),
	}, opts...)

	connector := NewStreamingConnector(endpoint, creds, connectorOpts...)
	m.connectors[endpoint] = connector

	return connector, nil
}

// Close closes all connectors.
func (m *ConnectorManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for endpoint, connector := range m.connectors {
		if err := connector.Close(); err != nil {
			m.log.Debug("Error closing connector", "endpoint", endpoint, "error", err)
		}
		delete(m.connectors, endpoint)
	}
	return nil
}

// ConnectorFactoryFunc is a function that creates a TypedExternalConnector.
type ConnectorFactoryFunc[T resource.Managed] func(endpoint string, useSSL bool, opts ...StreamingConnectorOption) managed.TypedExternalConnecter[T]

// SetupOptions holds the options for setting up a remote client.
type SetupOptions struct {
	// Endpoint is the gRPC endpoint of the provider.
	Endpoint string

	// UseSSL indicates whether to use SSL for the connection.
	UseSSL bool

	// Log is the logger to use.
	Log logging.Logger
}

// SetupOption configures SetupOptions.
type SetupOption func(*SetupOptions)

// WithEndpoint sets the gRPC endpoint.
func WithEndpoint(endpoint string) SetupOption {
	return func(o *SetupOptions) {
		o.Endpoint = endpoint
	}
}

// WithSSL enables or disables SSL for the connection.
func WithSSL(useSSL bool) SetupOption {
	return func(o *SetupOptions) {
		o.UseSSL = useSSL
	}
}

// WithSetupLogger sets the logger for setup operations.
func WithSetupLogger(log logging.Logger) SetupOption {
	return func(o *SetupOptions) {
		o.Log = log
	}
}

// DefaultSetupOptions returns the default options for setting up a remote client.
func DefaultSetupOptions() *SetupOptions {
	return &SetupOptions{
		Endpoint: "localhost:50051",
		UseSSL:   false,
		Log:      logging.NewNopLogger(),
	}
}

// SetupForResourceType creates a TypedExternalConnector factory for a specific resource type.
// The factory can then be used with managed.NewReconciler to create a reconciler that
// communicates with the remote provider.
func SetupForResourceType[T resource.Managed](mgr manager.Manager, opts ...SetupOption) ConnectorFactoryFunc[T] {
	// Process options
	options := DefaultSetupOptions()
	for _, opt := range opts {
		opt(options)
	}

	// Create a connector manager
	connectorManager := NewConnectorManager(options.Log)

	// Register the connector manager to be cleaned up when the manager stops
	mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		<-ctx.Done() // Wait for context to be cancelled (manager is stopping)
		return connectorManager.Close()
	}))

	// Return the factory function
	return func(endpoint string, useSSL bool, connectorOpts ...StreamingConnectorOption) managed.TypedExternalConnecter[T] {
		actualEndpoint := endpoint
		if actualEndpoint == "" {
			actualEndpoint = options.Endpoint
		}

		actualUseSSL := useSSL
		if !actualUseSSL {
			actualUseSSL = options.UseSSL
		}

		return &streamingConnectorAdapter[T]{
			manager:  connectorManager,
			endpoint: actualEndpoint,
			useSSL:   actualUseSSL,
			log:      options.Log,
			opts:     connectorOpts,
		}
	}
}

// streamingConnectorAdapter adapts a StreamingConnector to the TypedExternalConnector interface.
type streamingConnectorAdapter[T resource.Managed] struct {
	manager  *ConnectorManager
	endpoint string
	useSSL   bool
	log      logging.Logger
	opts     []StreamingConnectorOption
}

// Connect implements the TypedExternalConnector interface.
func (c *streamingConnectorAdapter[T]) Connect(ctx context.Context, mg T) (managed.TypedExternalClient[T], error) {
	// Determine credentials
	var creds credentials.TransportCredentials
	if c.useSSL {
		// In a real implementation, we'd load actual TLS credentials here
		// For now, we use insecure credentials for demonstration
		creds = insecure.NewCredentials()
	} else {
		creds = insecure.NewCredentials()
	}

	// Get the GVK for the resource
	gvk := mg.GetObjectKind().GroupVersionKind()

	// Get or create the connector
	connector, err := c.manager.GetOrCreateConnector(c.endpoint, creds, gvk, c.opts...)
	if err != nil {
		return nil, err
	}

	// Connect to the provider
	client, err := connector.Connect(ctx, mg)
	if err != nil {
		return nil, err
	}

	// Cast to the typed client
	return &typedClientAdapter[T]{client}, nil
}

// typedClientAdapter adapts an ExternalClient to the TypedExternalClient interface.
type typedClientAdapter[T resource.Managed] struct {
	client managed.ExternalClient
}

// Observe implements the TypedExternalClient interface.
func (c *typedClientAdapter[T]) Observe(ctx context.Context, mg T) (managed.ExternalObservation, error) {
	return c.client.Observe(ctx, mg)
}

// Create implements the TypedExternalClient interface.
func (c *typedClientAdapter[T]) Create(ctx context.Context, mg T) (managed.ExternalCreation, error) {
	return c.client.Create(ctx, mg)
}

// Update implements the TypedExternalClient interface.
func (c *typedClientAdapter[T]) Update(ctx context.Context, mg T) (managed.ExternalUpdate, error) {
	return c.client.Update(ctx, mg)
}

// Delete implements the TypedExternalClient interface.
func (c *typedClientAdapter[T]) Delete(ctx context.Context, mg T) (managed.ExternalDelete, error) {
	return c.client.Delete(ctx, mg)
}

// Disconnect implements the TypedExternalClient interface.
func (c *typedClientAdapter[T]) Disconnect(ctx context.Context) error {
	return c.client.Disconnect(ctx)
}