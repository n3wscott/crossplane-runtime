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

	"github.com/crossplane/crossplane-runtime/apis/proto/external/v1alpha1"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// A StreamingConnector produces a Client connected to a Server via a streaming gRPC connection.
type StreamingConnector struct {
	// conn is the underlying gRPC connection.
	conn *grpc.ClientConn

	// client is the gRPC service client.
	client v1alpha1.ExternalServiceClient

	// clientFactory is used to create gRPC clients.
	clientFactory GRPCClientFactory

	// endpoint is the gRPC server endpoint.
	endpoint string

	// credentials is the TLS credentials to use for the connection.
	credentials credentials.TransportCredentials

	// gvkMap maps from resource types to their TypedExternalConnector implementations.
	gvkMap map[schema.GroupVersionKind]struct{}

	// options contains additional gRPC dial options.
	options []grpc.DialOption

	// log is the logger to use.
	log logging.Logger

	// mu protects access to the connection.
	mu sync.Mutex
}

// A StreamingConnectorOption configures a StreamingConnector.
type StreamingConnectorOption func(*StreamingConnector)

// WithGRPCOptions configures the gRPC dial options.
func WithGRPCOptions(opts ...grpc.DialOption) StreamingConnectorOption {
	return func(c *StreamingConnector) {
		c.options = append(c.options, opts...)
	}
}

// WithGRPCClientFactory configures the gRPC client factory.
func WithGRPCClientFactory(factory GRPCClientFactory) StreamingConnectorOption {
	return func(c *StreamingConnector) {
		c.clientFactory = factory
	}
}

// WithClientLogger configures the logger for the client.
func WithClientLogger(log logging.Logger) StreamingConnectorOption {
	return func(c *StreamingConnector) {
		c.log = log
	}
}

// WithResourceTypes configures the types of resources that this connector can handle.
func WithResourceTypes(gvks ...schema.GroupVersionKind) StreamingConnectorOption {
	return func(c *StreamingConnector) {
		for _, gvk := range gvks {
			c.gvkMap[gvk] = struct{}{}
		}
	}
}

// NewStreamingConnector creates a StreamingConnector that produces clients connected to a Server.
func NewStreamingConnector(endpoint string, creds credentials.TransportCredentials, options ...StreamingConnectorOption) *StreamingConnector {
	c := &StreamingConnector{
		endpoint:      endpoint,
		credentials:   creds,
		clientFactory: &DefaultGRPCClientFactory{},
		gvkMap:        make(map[schema.GroupVersionKind]struct{}),
		log:           logging.NewNopLogger(),
		options: []grpc.DialOption{
			grpc.WithDefaultServiceConfig(lbRoundRobin),
		},
	}

	for _, o := range options {
		o(c)
	}

	// Add credentials to dial options if provided
	if creds != nil {
		c.options = append(c.options, grpc.WithTransportCredentials(creds))
	}

	return c
}

// Connect produces a StreamingClient connected to a Server via gRPC.
func (c *StreamingConnector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var err error

	// Get the GVK for the resource
	gvk := mg.GetObjectKind().GroupVersionKind()

	// Verify this connector can handle this resource type
	if len(c.gvkMap) > 0 {
		if _, ok := c.gvkMap[gvk]; !ok {
			return nil, errors.Errorf("connector does not support resource type %s", gvk)
		}
	}

	// Create the connection if it doesn't exist yet
	if c.conn == nil {
		c.conn, err = c.clientFactory.NewClient(c.endpoint, c.options...)
		if err != nil {
			return nil, errors.Wrap(err, errConnectFailed)
		}
		c.client = v1alpha1.NewExternalServiceClient(c.conn)
	}

	// Create a new streaming client
	client := &StreamingClient{
		client:   c.client,
		gvk:      gvk,
		log:      c.log.WithValues("gvk", gvk.String()),
		resource: mg,
	}

	// Start the session
	if err := client.startSession(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errSessionStartFailed)
	}

	return client, nil
}

// connectClient creates a direct connection to the gRPC server without creating a streaming client.
// This is useful for operations that don't require a full streaming session, like discovery.
func (c *StreamingConnector) connectClient(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var err error

	// Create the connection if it doesn't exist yet
	if c.conn == nil {
		c.conn, err = c.clientFactory.NewClient(c.endpoint, c.options...)
		if err != nil {
			return errors.Wrap(err, errConnectFailed)
		}
		c.client = v1alpha1.NewExternalServiceClient(c.conn)
	}

	return nil
}

// Discover calls the provider's Discover method to retrieve available resource types.
// It returns a slice of resource type descriptors that can be used to set up controllers.
func (c *StreamingConnector) Discover(ctx context.Context) ([]ResourceTypeDescriptor, error) {
	if err := c.connectClient(ctx); err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Create the connection if it doesn't exist yet
	if c.conn == nil {
		return nil, errors.New("not connected to provider, call Connect first")
	}

	// Call the Discover method
	resp, err := c.client.Discover(ctx, &v1alpha1.DiscoveryRequest{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to discover resource types")
	}

	// Process the response into resource type descriptors
	descriptors := make([]ResourceTypeDescriptor, 0)

	// Process reconcilers from the response
	for _, reconciler := range resp.Reconcilers {
		if reconciler.For != nil {
			descriptor := ResourceTypeDescriptor{
				APIVersion: reconciler.For.ApiVersion,
				Kind:       reconciler.For.Kind,
			}
			descriptors = append(descriptors, descriptor)
		}
	}

	return descriptors, nil
}

// Close closes the underlying gRPC connection.
func (c *StreamingConnector) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return errors.Wrap(err, errDisconnectFailed)
		}
		c.conn = nil
		c.client = nil
	}
	return nil
}
