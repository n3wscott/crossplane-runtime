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
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/crossplane/crossplane-runtime/apis/proto/external/v1alpha1"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/external/common"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

// Session handles bidirectional streaming for external resource management operations.
func (s *ProviderServer) Session(stream v1alpha1.ExternalService_SessionServer) error {
	log := s.log.WithValues("session", fmt.Sprintf("%p", stream))
	log.Debug("Starting new session")

	ctx := stream.Context()
	var (
		// We track connected clients by resource type
		connected      bool
		resourceType   schema.GroupVersionKind
		externalClient managed.TypedExternalClient[resource.Managed]
	)

	defer func() {
		if connected && externalClient != nil {
			if err := externalClient.Disconnect(ctx); err != nil {
				log.Debug("Error disconnecting client", "error", err)
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Receive request from client
		req, err := stream.Recv()
		if err != nil {
			log.Debug("Error receiving from stream", "error", err)
			return errors.Wrap(err, errStreamRecv)
		}

		log.Debug("Received request", "named", req.Named)

		// Process request based on operation type
		switch op := req.Op.(type) {
		case *v1alpha1.Request_Connect:
			// Convert the proto struct to a managed resource
			mg, gvk, err := fromProtoStruct(s.scheme, op.Connect.Resource)
			if err != nil {
				log.Debug("Error converting resource", "error", err)
				return err
			}

			log = log.WithValues("gvk", gvk.String())

			// Get the appropriate handler for this resource type
			s.mu.RLock()
			c, ok := s.handlers[gvk]
			s.mu.RUnlock()

			if !ok {
				log.Debug(errNoMatchingResourceTypeHandler)
				return errors.Errorf(errNoMatchingResourceTypeHandler)
			}

			// Connect to the external service
			client, err := c.Connect(ctx, mg)
			if err != nil {
				log.Debug("Error connecting to external service", "error", err)
				return err
			}

			// Save connection state
			resourceType = gvk
			externalClient = client
			connected = true

			// Convert the managed resource back to a proto struct after any possible modifications
			updatedStruct, err := common.AsStruct(mg)
			if err != nil {
				log.Debug("Error converting resource back to proto", "error", err)
				return err
			}

			// Create response with connection details (if any)
			resp := &v1alpha1.ConnectResponse{
				Resource: updatedStruct,
			}

			// Send response
			if err := stream.Send(&v1alpha1.Response{
				Op: &v1alpha1.Response_Connect{Connect: resp},
			}); err != nil {
				log.Debug("Error sending response", "error", err)
				return errors.Wrap(err, errStreamSend)
			}

		case *v1alpha1.Request_Observe:
			if !connected || externalClient == nil {
				log.Debug("Observe called before successful Connect")
				return errors.New("observe called before successful connect")
			}

			// Convert the proto struct to a managed resource
			mg, gvk, err := fromProtoStruct(s.scheme, op.Observe.Resource)
			if err != nil {
				log.Debug("Error converting resource", "error", err)
				return err
			}

			// Verify resource type matches connected type
			if gvk != resourceType {
				log.Debug("Resource type mismatch in Observe", "expected", resourceType, "got", gvk)
				return errors.Errorf("resource type mismatch: expected %s, got %s", resourceType, gvk)
			}

			// Observe the external resource
			observation, err := externalClient.Observe(ctx, mg)
			if err != nil {
				log.Debug("Error observing external resource", "error", err)
				return err
			}

			// Convert the managed resource back to a proto struct
			updatedStruct, err := common.AsStruct(mg)
			if err != nil {
				log.Debug("Error converting resource back to proto", "error", err)
				return err
			}

			// Create response
			resp := &v1alpha1.ObserveResponse{
				Resource:                updatedStruct,
				ConnectionDetails:       observation.ConnectionDetails,
				ResourceExists:          observation.ResourceExists,
				ResourceUpToDate:        observation.ResourceUpToDate,
				ResourceLateInitialized: observation.ResourceLateInitialized,
			}

			// Send response
			if err := stream.Send(&v1alpha1.Response{
				Op: &v1alpha1.Response_Observe{Observe: resp},
			}); err != nil {
				log.Debug("Error sending response", "error", err)
				return errors.Wrap(err, errStreamSend)
			}

		case *v1alpha1.Request_Create:
			if !connected || externalClient == nil {
				log.Debug("Create called before successful Connect")
				return errors.New("create called before successful connect")
			}

			// Convert the proto struct to a managed resource
			mg, gvk, err := fromProtoStruct(s.scheme, op.Create.Resource)
			if err != nil {
				log.Debug("Error converting resource", "error", err)
				return err
			}

			// Verify resource type matches connected type
			if gvk != resourceType {
				log.Debug("Resource type mismatch in Create", "expected", resourceType, "got", gvk)
				return errors.Errorf("resource type mismatch: expected %s, got %s", resourceType, gvk)
			}

			// Create the external resource
			creation, err := externalClient.Create(ctx, mg)
			if err != nil {
				log.Debug("Error creating external resource", "error", err)
				return err
			}

			// Convert the managed resource back to a proto struct
			updatedStruct, err := common.AsStruct(mg)
			if err != nil {
				log.Debug("Error converting resource back to proto", "error", err)
				return err
			}

			// Create response
			resp := &v1alpha1.CreateResponse{
				Resource:          updatedStruct,
				ConnectionDetails: creation.ConnectionDetails,
				AdditionalDetails: creation.AdditionalDetails,
			}

			// Send response
			if err := stream.Send(&v1alpha1.Response{
				Op: &v1alpha1.Response_Create{Create: resp},
			}); err != nil {
				log.Debug("Error sending response", "error", err)
				return errors.Wrap(err, errStreamSend)
			}

		case *v1alpha1.Request_Update:
			if !connected || externalClient == nil {
				log.Debug("Update called before successful Connect")
				return errors.New("update called before successful connect")
			}

			// Convert the proto struct to a managed resource
			mg, gvk, err := fromProtoStruct(s.scheme, op.Update.Resource)
			if err != nil {
				log.Debug("Error converting resource", "error", err)
				return err
			}

			// Verify resource type matches connected type
			if gvk != resourceType {
				log.Debug("Resource type mismatch in Update", "expected", resourceType, "got", gvk)
				return errors.Errorf("resource type mismatch: expected %s, got %s", resourceType, gvk)
			}

			// Update the external resource
			update, err := externalClient.Update(ctx, mg)
			if err != nil {
				log.Debug("Error updating external resource", "error", err)
				return err
			}

			// Convert the managed resource back to a proto struct
			updatedStruct, err := common.AsStruct(mg)
			if err != nil {
				log.Debug("Error converting resource back to proto", "error", err)
				return err
			}

			// Create response
			resp := &v1alpha1.UpdateResponse{
				Resource:          updatedStruct,
				ConnectionDetails: update.ConnectionDetails,
				AdditionalDetails: update.AdditionalDetails,
			}

			// Send response
			if err := stream.Send(&v1alpha1.Response{
				Op: &v1alpha1.Response_Update{Update: resp},
			}); err != nil {
				log.Debug("Error sending response", "error", err)
				return errors.Wrap(err, errStreamSend)
			}

		case *v1alpha1.Request_Delete:
			if !connected || externalClient == nil {
				log.Debug("Delete called before successful Connect")
				return errors.New("delete called before successful connect")
			}

			// Convert the proto struct to a managed resource
			mg, gvk, err := fromProtoStruct(s.scheme, op.Delete.Resource)
			if err != nil {
				log.Debug("Error converting resource", "error", err)
				return err
			}

			// Verify resource type matches connected type
			if gvk != resourceType {
				log.Debug("Resource type mismatch in Delete", "expected", resourceType, "got", gvk)
				return errors.Errorf("resource type mismatch: expected %s, got %s", resourceType, gvk)
			}

			// Delete the external resource
			deletion, err := externalClient.Delete(ctx, mg)
			if err != nil {
				log.Debug("Error deleting external resource", "error", err)
				return err
			}

			// Convert the managed resource back to a proto struct
			updatedStruct, err := common.AsStruct(mg)
			if err != nil {
				log.Debug("Error converting resource back to proto", "error", err)
				return err
			}

			// Create response
			resp := &v1alpha1.DeleteResponse{
				Resource:          updatedStruct,
				AdditionalDetails: deletion.AdditionalDetails,
			}

			// Send response
			if err := stream.Send(&v1alpha1.Response{
				Op: &v1alpha1.Response_Delete{Delete: resp},
			}); err != nil {
				log.Debug("Error sending response", "error", err)
				return errors.Wrap(err, errStreamSend)
			}

		case *v1alpha1.Request_Disconnect:
			// Disconnect the external client if connected
			if connected && externalClient != nil {
				if err := externalClient.Disconnect(ctx); err != nil {
					log.Debug("Error disconnecting from external service", "error", err)
					// Continue anyway as we want to clean up
				}
			}

			// Reset connection state
			connected = false
			externalClient = nil

			// Send response
			if err := stream.Send(&v1alpha1.Response{
				Op: &v1alpha1.Response_Disconnect{Disconnect: &v1alpha1.DisconnectResponse{}},
			}); err != nil {
				log.Debug("Error sending response", "error", err)
				return errors.Wrap(err, errStreamSend)
			}

		default:
			log.Debug(errNoMatchingOperation)
			return errors.New(errNoMatchingOperation)
		}
	}
}

// The fromProtoStruct function has been moved to util.go
