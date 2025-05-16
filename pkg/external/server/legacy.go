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
	"fmt"
	"sync"

	"github.com/crossplane/crossplane-runtime/apis/proto/external/v1alpha1"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/external/common"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// LegacyServer implements the legacy ConnectedExternalService API.
// This provides backward compatibility with the non-streaming gRPC API.
type LegacyServer struct {
	v1alpha1.UnimplementedConnectedExternalServiceServer

	// providerServer is the streaming server implementation
	providerServer *ProviderServer

	// log is the logger to use
	log logging.Logger

	// connections tracks connected client resources
	connections map[string]managed.ExternalClient

	// mu protects the connections map
	mu sync.RWMutex
}

// NewLegacyServer creates a new LegacyServer for the given ProviderServer.
func NewLegacyServer(providerServer *ProviderServer, log logging.Logger) *LegacyServer {
	return &LegacyServer{
		providerServer: providerServer,
		log:            log.WithValues("service", "legacy"),
		connections:    make(map[string]managed.ExternalClient),
	}
}

// RegisterWithServer registers the legacy service with the given gRPC server.
func (s *LegacyServer) RegisterWithServer(server *grpc.Server) {
	v1alpha1.RegisterConnectedExternalServiceServer(server, s)
}

// GetHandler returns a handler for the given resource GVK.
func (s *LegacyServer) GetHandler(ctx context.Context, gvk schema.GroupVersionKind) (managed.TypedExternalConnecter[resource.Managed], bool) {
	s.providerServer.mu.RLock()
	defer s.providerServer.mu.RUnlock()

	handler, ok := s.providerServer.handlers[gvk]
	return handler, ok
}

// Observe implements the legacy Observe method.
func (s *LegacyServer) Observe(ctx context.Context, req *v1alpha1.ObserveRequest) (*v1alpha1.ObserveResponse, error) {
	// Parse the resource and get its GVK
	mg, gvk, err := s.providerServer.fromProtoStruct(req.Resource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse resource")
	}

	// Generate a connection key
	connKey := fmt.Sprintf("%s/%p", gvk.String(), req.Resource)

	// Get a client for the resource
	client, err := s.getOrCreateClient(ctx, connKey, gvk, mg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get client")
	}

	// Call Observe
	observation, err := client.Observe(ctx, mg)
	if err != nil {
		return nil, errors.Wrap(err, "observe failed")
	}

	// Convert the managed resource back to a proto struct after any possible modifications
	updatedStruct, err := common.AsStruct(mg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert resource to proto struct")
	}

	// Build and return the response
	return &v1alpha1.ObserveResponse{
		Resource:                updatedStruct,
		ConnectionDetails:       observation.ConnectionDetails,
		ResourceExists:          observation.ResourceExists,
		ResourceUpToDate:        observation.ResourceUpToDate,
		ResourceLateInitialized: observation.ResourceLateInitialized,
	}, nil
}

// Create implements the legacy Create method.
func (s *LegacyServer) Create(ctx context.Context, req *v1alpha1.CreateRequest) (*v1alpha1.CreateResponse, error) {
	// Parse the resource and get its GVK
	mg, gvk, err := s.providerServer.fromProtoStruct(req.Resource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse resource")
	}

	// Generate a connection key
	connKey := fmt.Sprintf("%s/%p", gvk.String(), req.Resource)

	// Get a client for the resource
	client, err := s.getOrCreateClient(ctx, connKey, gvk, mg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get client")
	}

	// Call Create
	creation, err := client.Create(ctx, mg)
	if err != nil {
		return nil, errors.Wrap(err, "create failed")
	}

	// Convert the managed resource back to a proto struct after any possible modifications
	updatedStruct, err := common.AsStruct(mg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert resource to proto struct")
	}

	// Build and return the response
	return &v1alpha1.CreateResponse{
		Resource:          updatedStruct,
		ConnectionDetails: creation.ConnectionDetails,
		AdditionalDetails: creation.AdditionalDetails,
	}, nil
}

// Update implements the legacy Update method.
func (s *LegacyServer) Update(ctx context.Context, req *v1alpha1.UpdateRequest) (*v1alpha1.UpdateResponse, error) {
	// Parse the resource and get its GVK
	mg, gvk, err := s.providerServer.fromProtoStruct(req.Resource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse resource")
	}

	// Generate a connection key
	connKey := fmt.Sprintf("%s/%p", gvk.String(), req.Resource)

	// Get a client for the resource
	client, err := s.getOrCreateClient(ctx, connKey, gvk, mg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get client")
	}

	// Call Update
	update, err := client.Update(ctx, mg)
	if err != nil {
		return nil, errors.Wrap(err, "update failed")
	}

	// Convert the managed resource back to a proto struct after any possible modifications
	updatedStruct, err := common.AsStruct(mg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert resource to proto struct")
	}

	// Build and return the response
	return &v1alpha1.UpdateResponse{
		Resource:          updatedStruct,
		ConnectionDetails: update.ConnectionDetails,
		AdditionalDetails: update.AdditionalDetails,
	}, nil
}

// Delete implements the legacy Delete method.
func (s *LegacyServer) Delete(ctx context.Context, req *v1alpha1.DeleteRequest) (*v1alpha1.DeleteResponse, error) {
	// Parse the resource and get its GVK
	mg, gvk, err := s.providerServer.fromProtoStruct(req.Resource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse resource")
	}

	// Generate a connection key
	connKey := fmt.Sprintf("%s/%p", gvk.String(), req.Resource)

	// Get a client for the resource
	client, err := s.getOrCreateClient(ctx, connKey, gvk, mg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get client")
	}

	// Call Delete
	deletion, err := client.Delete(ctx, mg)
	if err != nil {
		return nil, errors.Wrap(err, "delete failed")
	}

	// Disconnect the client after delete
	s.disconnectClient(connKey)

	// Convert the managed resource back to a proto struct after any possible modifications
	updatedStruct, err := common.AsStruct(mg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert resource to proto struct")
	}

	// Build and return the response
	return &v1alpha1.DeleteResponse{
		Resource:          updatedStruct,
		AdditionalDetails: deletion.AdditionalDetails,
	}, nil
}

// getOrCreateClient gets an existing client or creates a new one for the given resource.
func (s *LegacyServer) getOrCreateClient(ctx context.Context, key string, gvk schema.GroupVersionKind, mg resource.Managed) (managed.ExternalClient, error) {
	// First check if we already have a client for this resource
	s.mu.RLock()
	client, exists := s.connections[key]
	s.mu.RUnlock()

	if exists {
		return client, nil
	}

	// If no client exists, we need to create one
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check in case another goroutine created it
	if client, exists = s.connections[key]; exists {
		return client, nil
	}

	// Get the handler for this resource type
	handler, ok := s.GetHandler(ctx, gvk)
	if !ok {
		return nil, errors.Errorf("no handler registered for resource type %s", gvk)
	}

	// Connect to the external resource
	client, err := handler.Connect(ctx, mg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to external resource")
	}

	// Store the client for future use
	s.connections[key] = client

	return client, nil
}

// disconnectClient disconnects and removes a client from the connections map.
func (s *LegacyServer) disconnectClient(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if client, exists := s.connections[key]; exists {
		// Call Disconnect, but don't fail if it doesn't work
		if err := client.Disconnect(context.Background()); err != nil {
			s.log.Debug("Error disconnecting client", "key", key, "error", err)
		}
		delete(s.connections, key)
	}
}

// CleanupConnections disconnects all clients and clears the connections map.
func (s *LegacyServer) CleanupConnections() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for key, client := range s.connections {
		// Call Disconnect, but don't fail if it doesn't work
		if err := client.Disconnect(context.Background()); err != nil {
			s.log.Debug("Error disconnecting client during cleanup", "key", key, "error", err)
		}
	}

	// Clear the map
	s.connections = make(map[string]managed.ExternalClient)
}