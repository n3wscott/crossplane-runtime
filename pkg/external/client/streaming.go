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
	"github.com/crossplane/crossplane-runtime/pkg/external/common"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// A StreamingClient uses a streaming gRPC session to communicate with a remote provider.
type StreamingClient struct {
	// client is the gRPC service client.
	client v1alpha1.ExternalServiceClient
	
	// stream is the bidirectional stream for this session.
	stream v1alpha1.ExternalService_SessionClient
	
	// mu protects the state of the session.
	mu sync.Mutex
	
	// gvk is the GroupVersionKind of the managed resource.
	gvk schema.GroupVersionKind
	
	// log is the logger to use.
	log logging.Logger
	
	// resource is the managed resource being managed.
	resource resource.Managed
}

// startSession starts a new bidirectional streaming session with the server.
func (c *StreamingClient) startSession(ctx context.Context, mg resource.Managed) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create the stream
	stream, err := c.client.Session(ctx)
	if err != nil {
		return errors.Wrap(err, errSessionStartFailed)
	}

	// Convert the managed resource to a struct
	s, err := common.AsStruct(mg)
	if err != nil {
		return errors.Wrap(err, errConvertManagedFailed)
	}

	// Send the connect request
	req := &v1alpha1.Request{
		Named: c.gvk.String(),
		Op: &v1alpha1.Request_Connect{
			Connect: &v1alpha1.ConnectRequest{
				Resource: s,
			},
		},
	}

	if err := stream.Send(req); err != nil {
		return errors.Wrap(err, errSendRequestFailed)
	}

	// Receive the connect response
	resp, err := stream.Recv()
	if err != nil {
		return errors.Wrap(err, errRecvResponseFailed)
	}

	// Check for the correct response type
	connectResp, ok := resp.Op.(*v1alpha1.Response_Connect)
	if !ok {
		return errors.New(errInvalidResponseType)
	}

	// Update the managed resource with any changes from the server
	if err := common.AsManaged(connectResp.Connect.Resource, mg); err != nil {
		return errors.Wrap(err, errConvertManagedFailed)
	}

	// Store the stream for future operations
	c.stream = stream
	c.resource = mg

	return nil
}

// Observe the supplied managed resource.
func (c *StreamingClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Convert the managed resource to a struct
	s, err := common.AsStruct(mg)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errConvertManagedFailed)
	}

	// Send the observe request
	req := &v1alpha1.Request{
		Named: c.gvk.String(),
		Op: &v1alpha1.Request_Observe{
			Observe: &v1alpha1.ObserveRequest{
				Resource: s,
			},
		},
	}

	if err := c.stream.Send(req); err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errSendRequestFailed)
	}

	// Receive the observe response
	resp, err := c.stream.Recv()
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errRecvResponseFailed)
	}

	// Check for the correct response type
	observeResp, ok := resp.Op.(*v1alpha1.Response_Observe)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errInvalidResponseType)
	}

	// Update the managed resource with the response
	if err := common.AsManaged(observeResp.Observe.Resource, mg); err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errConvertManagedFailed)
	}

	return managed.ExternalObservation{
		ResourceExists:          observeResp.Observe.ResourceExists,
		ResourceUpToDate:        observeResp.Observe.ResourceUpToDate,
		ResourceLateInitialized: observeResp.Observe.ResourceLateInitialized,
		ConnectionDetails:       observeResp.Observe.ConnectionDetails,
	}, nil
}

// Create the supplied managed resource.
func (c *StreamingClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Convert the managed resource to a struct
	s, err := common.AsStruct(mg)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errConvertManagedFailed)
	}

	// Send the create request
	req := &v1alpha1.Request{
		Named: c.gvk.String(),
		Op: &v1alpha1.Request_Create{
			Create: &v1alpha1.CreateRequest{
				Resource: s,
			},
		},
	}

	if err := c.stream.Send(req); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errSendRequestFailed)
	}

	// Receive the create response
	resp, err := c.stream.Recv()
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errRecvResponseFailed)
	}

	// Check for the correct response type
	createResp, ok := resp.Op.(*v1alpha1.Response_Create)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errInvalidResponseType)
	}

	// Update the managed resource with the response
	if err := common.AsManaged(createResp.Create.Resource, mg); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errConvertManagedFailed)
	}

	return managed.ExternalCreation{
		ConnectionDetails: createResp.Create.ConnectionDetails,
		AdditionalDetails: createResp.Create.AdditionalDetails,
	}, nil
}

// Update the supplied managed resource.
func (c *StreamingClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Convert the managed resource to a struct
	s, err := common.AsStruct(mg)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errConvertManagedFailed)
	}

	// Send the update request
	req := &v1alpha1.Request{
		Named: c.gvk.String(),
		Op: &v1alpha1.Request_Update{
			Update: &v1alpha1.UpdateRequest{
				Resource: s,
			},
		},
	}

	if err := c.stream.Send(req); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errSendRequestFailed)
	}

	// Receive the update response
	resp, err := c.stream.Recv()
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errRecvResponseFailed)
	}

	// Check for the correct response type
	updateResp, ok := resp.Op.(*v1alpha1.Response_Update)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errInvalidResponseType)
	}

	// Update the managed resource with the response
	if err := common.AsManaged(updateResp.Update.Resource, mg); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errConvertManagedFailed)
	}

	return managed.ExternalUpdate{
		ConnectionDetails: updateResp.Update.ConnectionDetails,
		AdditionalDetails: updateResp.Update.AdditionalDetails,
	}, nil
}

// Delete the supplied managed resource.
func (c *StreamingClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Convert the managed resource to a struct
	s, err := common.AsStruct(mg)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errConvertManagedFailed)
	}

	// Send the delete request
	req := &v1alpha1.Request{
		Named: c.gvk.String(),
		Op: &v1alpha1.Request_Delete{
			Delete: &v1alpha1.DeleteRequest{
				Resource: s,
			},
		},
	}

	if err := c.stream.Send(req); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errSendRequestFailed)
	}

	// Receive the delete response
	resp, err := c.stream.Recv()
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errRecvResponseFailed)
	}

	// Check for the correct response type
	deleteResp, ok := resp.Op.(*v1alpha1.Response_Delete)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errInvalidResponseType)
	}

	// Update the managed resource with the response
	if err := common.AsManaged(deleteResp.Delete.Resource, mg); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errConvertManagedFailed)
	}

	return managed.ExternalDelete{
		AdditionalDetails: deleteResp.Delete.AdditionalDetails,
	}, nil
}

// Disconnect closes the gRPC session.
func (c *StreamingClient) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stream == nil {
		return nil
	}

	// Send the disconnect request
	req := &v1alpha1.Request{
		Named: c.gvk.String(),
		Op: &v1alpha1.Request_Disconnect{
			Disconnect: &v1alpha1.DisconnectRequest{},
		},
	}

	if err := c.stream.Send(req); err != nil {
		c.log.Debug("Error sending disconnect request", "error", err)
		// Continue anyway as we're closing the connection
	}

	// Try to receive the disconnect response, but don't fail if we can't
	resp, err := c.stream.Recv()
	if err != nil {
		c.log.Debug("Error receiving disconnect response", "error", err)
		// Continue anyway as we're closing the connection
	} else {
		// Verify we got a disconnect response
		if _, ok := resp.Op.(*v1alpha1.Response_Disconnect); !ok {
			c.log.Debug("Received unexpected response type during disconnect")
		}
	}

	c.stream = nil
	return nil
}