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

package remote

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/crossplane/crossplane-runtime/apis/proto/external/v1alpha1"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

// TODO(negz): Should any of these be configurable?
const (
	// This configures a gRPC client to use round robin load balancing.
	// See https://github.com/grpc/grpc/blob/v1.58.0/doc/load-balancing.md#load-balancing-policies
	lbRoundRobin = `{"loadBalancingConfig":[{"round_robin":{}}]}`
)

// A Connector produces a Client connected to a Server via gRPC. Unlike most
// ExternalConnector implementations it doesn't create a new connection each
// time it's called, but instead reuses the same gRPC client connection.
type Connector struct {
	sc v1alpha1.ExternalServiceClient
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

	return &Connector{sc: v1alpha1.NewExternalServiceClient(conn)}, nil
}

// Connect produces a Client connected to a Server via gRPC. Unlike most
// ExternalConnector implementations it doesn't create a new connection each
// time it's called, but instead reuses the same gRPC client connection.
func (c *Connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	return &Client{sc: c.sc}, nil
}

// A Client uses a Server to observe, create, update, and delete external
// resources.
type Client struct {
	sc v1alpha1.ExternalServiceClient
}

// Observe the supplied managed resource.
func (c *Client) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	s, err := AsStruct(mg)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	rsp, err := c.sc.Observe(ctx, &v1alpha1.ObserveRequest{Resource: s})
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	if err := AsManaged(rsp.GetResource(), mg); err != nil {
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
	s, err := AsStruct(mg)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	rsp, err := c.sc.Create(ctx, &v1alpha1.CreateRequest{Resource: s})
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if err := AsManaged(rsp.GetResource(), mg); err != nil {
		return managed.ExternalCreation{}, err
	}

	return managed.ExternalCreation{
		ConnectionDetails: rsp.GetConnectionDetails(),
		AdditionalDetails: rsp.AdditionalDetails,
	}, nil
}

// Update the supplied managed resource.
func (c *Client) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	s, err := AsStruct(mg)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	rsp, err := c.sc.Update(ctx, &v1alpha1.UpdateRequest{Resource: s})
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	if err := AsManaged(rsp.GetResource(), mg); err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{
		ConnectionDetails: rsp.GetConnectionDetails(),
		AdditionalDetails: rsp.AdditionalDetails,
	}, nil
}

// Delete the supplied managed resource.
func (c *Client) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	s, err := AsStruct(mg)
	if err != nil {
		return managed.ExternalDelete{}, err
	}

	rsp, err := c.sc.Delete(ctx, &v1alpha1.DeleteRequest{Resource: s})
	if err != nil {
		return managed.ExternalDelete{}, err
	}

	return managed.ExternalDelete{AdditionalDetails: rsp.AdditionalDetails}, AsManaged(rsp.GetResource(), mg)
}

func (c *Client) Disconnect(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}
