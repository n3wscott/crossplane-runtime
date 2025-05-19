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

package server

import (
	"context"
	"net"
	"os"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

// ProviderServerConfig contains the configuration for the gRPC provider server.
type ProviderServerConfig struct {
	// Address is the gRPC server address to listen on.
	Address string

	// TLSCertPath is the path to the TLS certificate.
	TLSCertPath string

	// TLSKeyPath is the path to the TLS key.
	TLSKeyPath string

	// Log is the logger to use.
	Log logging.Logger

	// GRPCServerOptions are additional options for the gRPC server.
	GRPCServerOptions []grpc.ServerOption
}

// ProviderOption configures a ProviderServerConfig.
type ProviderOption func(*ProviderServerConfig)

// WithProviderAddress sets the server address.
func WithProviderAddress(address string) ProviderOption {
	return func(c *ProviderServerConfig) {
		c.Address = address
	}
}

// WithProviderTLSCertPath sets the TLS certificate path.
func WithProviderTLSCertPath(certPath string) ProviderOption {
	return func(c *ProviderServerConfig) {
		c.TLSCertPath = certPath
	}
}

// WithProviderTLSKeyPath sets the TLS key path.
func WithProviderTLSKeyPath(keyPath string) ProviderOption {
	return func(c *ProviderServerConfig) {
		c.TLSKeyPath = keyPath
	}
}

// WithProviderLogger sets the logger for the server.
func WithProviderLogger(log logging.Logger) ProviderOption {
	return func(c *ProviderServerConfig) {
		c.Log = log
	}
}

// WithProviderGRPCOptions adds additional gRPC server options.
func WithProviderGRPCOptions(opts ...grpc.ServerOption) ProviderOption {
	return func(c *ProviderServerConfig) {
		c.GRPCServerOptions = append(c.GRPCServerOptions, opts...)
	}
}

// ProviderBuilder is used to simplify the registration of handlers for a provider.
type ProviderBuilder struct {
	// health is a gRPC health server implementation.
	health grpc_health_v1.HealthServer

	// server is the underlying ProviderServer.
	server *ProviderServer

	// config holds the server configuration.
	config *ProviderServerConfig

	// grpcServer is the actual gRPC server.
	grpcServer *grpc.Server

	// listener is the network listener.
	listener net.Listener

	// log is the logger to use.
	log logging.Logger
}

// NewProviderBuilder creates a new ProviderBuilder.
func NewProviderBuilder(scheme *runtime.Scheme, opts ...ProviderOption) (*ProviderBuilder, error) {
	// Default configuration
	config := &ProviderServerConfig{
		Address: ":50051",
		Log:     logging.NewNopLogger(),
	}

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	// Configure logging
	log := config.Log.WithValues("component", "grpc-provider")

	// Create server options for the underlying ProviderServer
	var serverOpts []ProviderServerOption
	// Use WithServerLogger from server.go
	serverOpts = append(serverOpts, WithServerLogger(log))

	// If TLS is configured, add TLS credentials
	if config.TLSCertPath != "" && config.TLSKeyPath != "" {
		creds, err := credentials.NewServerTLSFromFile(config.TLSCertPath, config.TLSKeyPath)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load TLS credentials")
		}
		serverOpts = append(serverOpts, WithTLSCredentials(creds))
	}

	// Create the gRPC server options
	var grpcOpts []grpc.ServerOption
	grpcOpts = append(grpcOpts, config.GRPCServerOptions...)

	// Create the gRPC server
	grpcServer := grpc.NewServer(grpcOpts...)

	// Create the provider server
	server := NewProviderServer(scheme, serverOpts...)

	return &ProviderBuilder{
		server:     server,
		config:     config,
		grpcServer: grpcServer,
		log:        log,
	}, nil
}

// RegisterHandler registers a handler for a specific resource type.
func (b *ProviderBuilder) RegisterHandler(gvk schema.GroupVersionKind, handler managed.TypedExternalConnecter[resource.Managed]) error {
	return b.server.RegisterHandler(gvk, handler)
}

// RegisterHandlerFunc registers a handler function for a specific resource type.
// It creates a wrapper that adapts the function to the TypedExternalConnector interface.
func (b *ProviderBuilder) RegisterHandlerFunc(gvk schema.GroupVersionKind, connectFunc func(ctx context.Context, mg resource.Managed) (managed.TypedExternalClient[resource.Managed], error)) error {
	wrapper := &ConnectorFuncWrapper{
		ConnectFunc: connectFunc,
	}
	return b.server.RegisterHandler(gvk, wrapper)
}

// RegisterExistingConnector registers an existing ExternalConnector for a specific resource type.
// It creates a wrapper that adapts the ExternalConnector to the TypedExternalConnector interface.
func (b *ProviderBuilder) RegisterExistingConnector(gvk schema.GroupVersionKind, connector managed.ExternalConnecter) error {
	wrapper := &ExternalConnectorWrapper{
		Connector: connector,
	}
	return b.server.RegisterHandler(gvk, wrapper)
}

// Start starts the gRPC server.
func (b *ProviderBuilder) Start(ctx context.Context) error {
	if b.health == nil {
		b.health = health.NewServer()
	}
	grpc_health_v1.RegisterHealthServer(b.grpcServer, b.health)

	// Register the provider server with the gRPC server
	b.server.RegisterWithServer(b.grpcServer)

	// TODO: we could have an option to not add the reflection server.
	reflection.Register(b.grpcServer)

	var err error
	b.listener, err = net.Listen("tcp", b.config.Address)
	if err != nil {
		return errors.Wrap(err, "failed to create listener")
	}

	b.log.Info("Starting gRPC provider server", "address", b.config.Address)

	// Try casting to a default health server and set the serving status.
	if healthServer, ok := b.health.(*health.Server); ok {
		// TODO: we could look at what is being served and host it inside of the serving status for each kind.
		healthServer.SetServingStatus("provider-service", grpc_health_v1.HealthCheckResponse_SERVING)
	}

	// Start serving in a goroutine
	go func() {
		if err := b.grpcServer.Serve(b.listener); err != nil {
			b.log.Info("Failed to serve gRPC", "error", err)
		}
	}()

	// Wait for context cancellation to stop the server
	go func() {
		<-ctx.Done()
		b.log.Info("Stopping gRPC provider server")
		b.grpcServer.GracefulStop()
	}()

	return nil
}

// Stop stops the gRPC server.
func (b *ProviderBuilder) Stop() {
	b.grpcServer.GracefulStop()
	if b.listener != nil {
		b.listener.Close()
	}
}

// ConnectorFuncWrapper wraps a connect function to implement TypedExternalConnector.
type ConnectorFuncWrapper struct {
	ConnectFunc func(ctx context.Context, mg resource.Managed) (managed.TypedExternalClient[resource.Managed], error)
}

// Connect implements the TypedExternalConnector interface.
func (w *ConnectorFuncWrapper) Connect(ctx context.Context, mg resource.Managed) (managed.TypedExternalClient[resource.Managed], error) {
	return w.ConnectFunc(ctx, mg)
}

// ExternalConnectorWrapper wraps an ExternalConnector to implement TypedExternalConnector.
type ExternalConnectorWrapper struct {
	Connector managed.ExternalConnecter
}

// Connect implements the TypedExternalConnector interface.
func (w *ExternalConnectorWrapper) Connect(ctx context.Context, mg resource.Managed) (managed.TypedExternalClient[resource.Managed], error) {
	client, err := w.Connector.Connect(ctx, mg)
	if err != nil {
		return nil, err
	}
	return &ExternalClientWrapper{Client: client}, nil
}

// ExternalClientWrapper wraps an ExternalClient to implement TypedExternalClient.
type ExternalClientWrapper struct {
	Client managed.ExternalClient
}

// Observe implements the TypedExternalClient interface.
func (w *ExternalClientWrapper) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	return w.Client.Observe(ctx, mg)
}

// Create implements the TypedExternalClient interface.
func (w *ExternalClientWrapper) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	return w.Client.Create(ctx, mg)
}

// Update implements the TypedExternalClient interface.
func (w *ExternalClientWrapper) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	return w.Client.Update(ctx, mg)
}

// Delete implements the TypedExternalClient interface.
func (w *ExternalClientWrapper) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	return w.Client.Delete(ctx, mg)
}

// Disconnect implements the TypedExternalClient interface.
func (w *ExternalClientWrapper) Disconnect(ctx context.Context) error {
	return w.Client.Disconnect(ctx)
}

// SetupProviderServer sets up a gRPC provider server and registers it with the manager.
// It looks for environment variables to configure the server:
// - GRPC_SERVER_ENABLED: If set to "true", the server will be enabled (default: "false")
// - GRPC_SERVER_ADDRESS: The address to listen on (default: ":50051")
// - GRPC_SERVER_TLS_CERT: Path to the TLS certificate (optional)
// - GRPC_SERVER_TLS_KEY: Path to the TLS key (optional)
func SetupProviderServer(mgr manager.Manager, registerFunc func(*ProviderBuilder) error) error {
	// Check if the server should be enabled
	enabled := strings.ToLower(os.Getenv("GRPC_SERVER_ENABLED")) == "true"
	if !enabled {
		ctrl.Log.Info("gRPC provider server is disabled")
		return nil
	}

	// Configure the provider
	opts := []ProviderOption{
		WithProviderLogger(logging.NewLogrLogger(ctrl.Log)),
	}

	// Set the address from environment variable if present
	if addr := os.Getenv("GRPC_SERVER_ADDRESS"); addr != "" {
		opts = append(opts, WithProviderAddress(addr))
	}

	// Set TLS certificate and key if present
	if certPath := os.Getenv("GRPC_SERVER_TLS_CERT"); certPath != "" {
		opts = append(opts, WithProviderTLSCertPath(certPath))
	}
	if keyPath := os.Getenv("GRPC_SERVER_TLS_KEY"); keyPath != "" {
		opts = append(opts, WithProviderTLSKeyPath(keyPath))
	}

	// Create the provider builder
	builder, err := NewProviderBuilder(mgr.GetScheme(), opts...)
	if err != nil {
		return errors.Wrap(err, "failed to create provider builder")
	}

	// Register handlers
	if err := registerFunc(builder); err != nil {
		return errors.Wrap(err, "failed to register handlers")
	}

	// Add the server to the manager
	mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		return builder.Start(ctx)
	}))

	return nil
}
