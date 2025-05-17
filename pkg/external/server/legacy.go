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

	"github.com/crossplane/crossplane-runtime/apis/proto/external/v1alpha1"
	"github.com/crossplane/crossplane-runtime/pkg/external/common"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/runtime"
)

// LegacyServer implements the non-streaming ConnectedExternalServiceServer API.
// It provides backward compatibility for clients that don't support the streaming API.
type LegacyServer struct {
	v1alpha1.UnimplementedConnectedExternalServiceServer

	// handlers maps resource types to their respective handlers.
	handlers TypeHandlerMap

	// scheme is used for managed resource encoding/decoding.
	scheme *runtime.Scheme

	// log is the server logger.
	log logging.Logger
}

// NewLegacyServer creates a new LegacyServer with the given options.
func NewLegacyServer(scheme *runtime.Scheme, handlers TypeHandlerMap, log logging.Logger) *LegacyServer {
	return &LegacyServer{
		handlers: handlers,
		scheme:   scheme,
		log:      log,
	}
}

// Observe the external resource the supplied managed resource represents.
func (s *LegacyServer) Observe(ctx context.Context, request *v1alpha1.ObserveRequest) (*v1alpha1.ObserveResponse, error) {
	// Convert the proto struct to a managed resource
	mg, gvk, err := fromProtoStruct(s.scheme, request.Resource)
	if err != nil {
		s.log.Debug("Error converting resource", "error", err)
		return nil, err
	}

	log := s.log.WithValues("gvk", gvk.String())

	// Get the appropriate handler for this resource type
	c, ok := s.handlers[gvk]
	if !ok {
		log.Debug(errNoMatchingResourceTypeHandler)
		return nil, errors.New(errNoMatchingResourceTypeHandler)
	}

	// Connect to the external service
	client, err := c.Connect(ctx, mg)
	if err != nil {
		log.Debug("Error connecting to external service", "error", err)
		return nil, err
	}

	// Ensure client is disconnected when done
	defer func() {
		if typed, ok := client.(interface{ Disconnect(context.Context) error }); ok {
			if err := typed.Disconnect(ctx); err != nil {
				log.Debug("Error disconnecting client", "error", err)
			}
		}
	}()

	// Observe the external resource
	observation, err := client.Observe(ctx, mg)
	if err != nil {
		log.Debug("Error observing external resource", "error", err)
		return nil, err
	}

	// Convert the managed resource back to a proto struct
	updatedStruct, err := common.AsStruct(mg)
	if err != nil {
		log.Debug("Error converting resource back to proto", "error", err)
		return nil, err
	}

	// Create response
	resp := &v1alpha1.ObserveResponse{
		Resource:                updatedStruct,
		ConnectionDetails:       observation.ConnectionDetails,
		ResourceExists:          observation.ResourceExists,
		ResourceUpToDate:        observation.ResourceUpToDate,
		ResourceLateInitialized: observation.ResourceLateInitialized,
	}

	return resp, nil
}

// Create an external resource per the specifications of the supplied managed resource.
func (s *LegacyServer) Create(ctx context.Context, request *v1alpha1.CreateRequest) (*v1alpha1.CreateResponse, error) {
	// Convert the proto struct to a managed resource
	mg, gvk, err := fromProtoStruct(s.scheme, request.Resource)
	if err != nil {
		s.log.Debug("Error converting resource", "error", err)
		return nil, err
	}

	log := s.log.WithValues("gvk", gvk.String())

	// Get the appropriate handler for this resource type
	c, ok := s.handlers[gvk]
	if !ok {
		log.Debug(errNoMatchingResourceTypeHandler)
		return nil, errors.New(errNoMatchingResourceTypeHandler)
	}

	// Connect to the external service
	client, err := c.Connect(ctx, mg)
	if err != nil {
		log.Debug("Error connecting to external service", "error", err)
		return nil, err
	}

	// Ensure client is disconnected when done
	defer func() {
		if typed, ok := client.(interface{ Disconnect(context.Context) error }); ok {
			if err := typed.Disconnect(ctx); err != nil {
				log.Debug("Error disconnecting client", "error", err)
			}
		}
	}()

	// Create the external resource
	creation, err := client.Create(ctx, mg)
	if err != nil {
		log.Debug("Error creating external resource", "error", err)
		return nil, err
	}

	// Convert the managed resource back to a proto struct
	updatedStruct, err := common.AsStruct(mg)
	if err != nil {
		log.Debug("Error converting resource back to proto", "error", err)
		return nil, err
	}

	// Create response
	resp := &v1alpha1.CreateResponse{
		Resource:          updatedStruct,
		ConnectionDetails: creation.ConnectionDetails,
		AdditionalDetails: creation.AdditionalDetails,
	}

	return resp, nil
}

// Update the external resource represented by the supplied managed resource.
func (s *LegacyServer) Update(ctx context.Context, request *v1alpha1.UpdateRequest) (*v1alpha1.UpdateResponse, error) {
	// Convert the proto struct to a managed resource
	mg, gvk, err := fromProtoStruct(s.scheme, request.Resource)
	if err != nil {
		s.log.Debug("Error converting resource", "error", err)
		return nil, err
	}

	log := s.log.WithValues("gvk", gvk.String())

	// Get the appropriate handler for this resource type
	c, ok := s.handlers[gvk]
	if !ok {
		log.Debug(errNoMatchingResourceTypeHandler)
		return nil, errors.New(errNoMatchingResourceTypeHandler)
	}

	// Connect to the external service
	client, err := c.Connect(ctx, mg)
	if err != nil {
		log.Debug("Error connecting to external service", "error", err)
		return nil, err
	}

	// Ensure client is disconnected when done
	defer func() {
		if typed, ok := client.(interface{ Disconnect(context.Context) error }); ok {
			if err := typed.Disconnect(ctx); err != nil {
				log.Debug("Error disconnecting client", "error", err)
			}
		}
	}()

	// Update the external resource
	update, err := client.Update(ctx, mg)
	if err != nil {
		log.Debug("Error updating external resource", "error", err)
		return nil, err
	}

	// Convert the managed resource back to a proto struct
	updatedStruct, err := common.AsStruct(mg)
	if err != nil {
		log.Debug("Error converting resource back to proto", "error", err)
		return nil, err
	}

	// Create response
	resp := &v1alpha1.UpdateResponse{
		Resource:          updatedStruct,
		ConnectionDetails: update.ConnectionDetails,
		AdditionalDetails: update.AdditionalDetails,
	}

	return resp, nil
}

// Delete the external resource upon deletion of its associated managed resource.
func (s *LegacyServer) Delete(ctx context.Context, request *v1alpha1.DeleteRequest) (*v1alpha1.DeleteResponse, error) {
	// Convert the proto struct to a managed resource
	mg, gvk, err := fromProtoStruct(s.scheme, request.Resource)
	if err != nil {
		s.log.Debug("Error converting resource", "error", err)
		return nil, err
	}

	log := s.log.WithValues("gvk", gvk.String())

	// Get the appropriate handler for this resource type
	c, ok := s.handlers[gvk]
	if !ok {
		log.Debug(errNoMatchingResourceTypeHandler)
		return nil, errors.New(errNoMatchingResourceTypeHandler)
	}

	// Connect to the external service
	client, err := c.Connect(ctx, mg)
	if err != nil {
		log.Debug("Error connecting to external service", "error", err)
		return nil, err
	}

	// Ensure client is disconnected when done
	defer func() {
		if typed, ok := client.(interface{ Disconnect(context.Context) error }); ok {
			if err := typed.Disconnect(ctx); err != nil {
				log.Debug("Error disconnecting client", "error", err)
			}
		}
	}()

	// Delete the external resource
	deletion, err := client.Delete(ctx, mg)
	if err != nil {
		log.Debug("Error deleting external resource", "error", err)
		return nil, err
	}

	// Convert the managed resource back to a proto struct
	updatedStruct, err := common.AsStruct(mg)
	if err != nil {
		log.Debug("Error converting resource back to proto", "error", err)
		return nil, err
	}

	// Create response
	resp := &v1alpha1.DeleteResponse{
		Resource:          updatedStruct,
		AdditionalDetails: deletion.AdditionalDetails,
	}

	return resp, nil
}

// RegisterWithServer registers both the streaming and legacy APIs with the given gRPC server.
func (s *LegacyServer) RegisterWithServer(server *grpc.Server) {
	v1alpha1.RegisterConnectedExternalServiceServer(server, s)
}
