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

// Package client provides client implementations for communicating with external providers.
package client

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/crossplane/crossplane-runtime/apis/proto/external/v1alpha1"
	"github.com/crossplane/crossplane-runtime/pkg/external/common"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

// ResourceTypeDescriptor describes a resource type that can be reconciled by a provider.
type ResourceTypeDescriptor struct {
	// APIVersion is the API version of the resource type.
	APIVersion string
	
	// Kind is the kind of the resource type.
	Kind string
}

// Error strings.
const (
	errConnectFailed        = "failed to connect to remote provider"
	errDisconnectFailed     = "failed to disconnect from remote provider"
	errStreamClosed         = "stream closed unexpectedly"
	errInvalidResponseType  = "received invalid response type from server"
	errNoResponse           = "no response received from server"
	errSessionStartFailed   = "failed to start gRPC session"
	errSessionCloseFailed   = "failed to close gRPC session"
	errSendRequestFailed    = "failed to send request to server"
	errRecvResponseFailed   = "failed to receive response from server"
	errConvertManagedFailed = "failed to convert managed resource"
)

// TODO(negz): Should any of these be configurable?
const (
	// This configures a gRPC client to use round robin load balancing.
	// See https://github.com/grpc/grpc/blob/v1.58.0/doc/load-balancing.md#load-balancing-policies
	lbRoundRobin = `{"loadBalancingConfig":[{"round_robin":{}}]}`
)

// A GRPCClientFactory creates gRPC clients for different endpoints.
type GRPCClientFactory interface {
	// NewClient creates a new gRPC client for the given endpoint.
	NewClient(endpoint string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
}

// DefaultGRPCClientFactory is the default implementation of GRPCClientFactory.
type DefaultGRPCClientFactory struct{}

// NewClient creates a new gRPC client for the given endpoint.
func (f *DefaultGRPCClientFactory) NewClient(endpoint string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return grpc.NewClient(endpoint, opts...)
}

// LegacySupport for non-streaming API (kept for backwards compatibility)
// ---------------------------------------------------------------------

// A Connector produces a Client connected to a Server via gRPC.
type Connector struct {
	sc v1alpha1.ConnectedExternalServiceClient
}

// NewConnector creates a Connector that produces clients connected to a Server
// running at the supplied gRPC endpoint.
func NewConnector(ctx context.Context, endpoint string, creds credentials.TransportCredentials) (*Connector, error) {
	conn, err := grpc.NewClient(endpoint,
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultServiceConfig(lbRoundRobin))
	if err != nil {
		return nil, err
	}

	return &Connector{sc: v1alpha1.NewConnectedExternalServiceClient(conn)}, nil
}

// Connect produces a Client connected to a Server via gRPC.
func (c *Connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	return &Client{sc: c.sc}, nil
}

// A Client uses a Server to observe, create, update, and delete external
// resources.
type Client struct {
	sc v1alpha1.ConnectedExternalServiceClient
}

// Observe the supplied managed resource.
func (c *Client) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	s, err := common.AsStruct(mg)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	rsp, err := c.sc.Observe(ctx, &v1alpha1.ObserveRequest{Resource: s})
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	if err := common.AsManaged(rsp.GetResource(), mg); err != nil {
		return managed.ExternalObservation{}, err
	}

	o := managed.ExternalObservation{
		ResourceExists:          rsp.GetResourceExists(),
		ResourceUpToDate:        rsp.GetResourceUpToDate(),
		ResourceLateInitialized: rsp.GetResourceLateInitialized(),
		ConnectionDetails:       rsp.GetConnectionDetails(),
	}

	return o, nil
}

// Create the supplied managed resource.
func (c *Client) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	s, err := common.AsStruct(mg)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	rsp, err := c.sc.Create(ctx, &v1alpha1.CreateRequest{Resource: s})
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if err := common.AsManaged(rsp.GetResource(), mg); err != nil {
		return managed.ExternalCreation{}, err
	}

	return managed.ExternalCreation{
		ConnectionDetails: rsp.GetConnectionDetails(),
		AdditionalDetails: rsp.AdditionalDetails,
	}, nil
}

// Update the supplied managed resource.
func (c *Client) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	s, err := common.AsStruct(mg)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	rsp, err := c.sc.Update(ctx, &v1alpha1.UpdateRequest{Resource: s})
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	if err := common.AsManaged(rsp.GetResource(), mg); err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{
		ConnectionDetails: rsp.GetConnectionDetails(),
		AdditionalDetails: rsp.AdditionalDetails,
	}, nil
}

// Delete the supplied managed resource.
func (c *Client) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	s, err := common.AsStruct(mg)
	if err != nil {
		return managed.ExternalDelete{}, err
	}

	rsp, err := c.sc.Delete(ctx, &v1alpha1.DeleteRequest{Resource: s})
	if err != nil {
		return managed.ExternalDelete{}, err
	}

	if err := common.AsManaged(rsp.GetResource(), mg); err != nil {
		return managed.ExternalDelete{}, err
	}

	return managed.ExternalDelete{AdditionalDetails: rsp.AdditionalDetails}, nil
}

// Disconnect closes the connection.
func (c *Client) Disconnect(ctx context.Context) error {
	// No explicit connection closing for the legacy client
	return nil
}
