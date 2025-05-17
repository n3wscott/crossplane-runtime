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

// Package server provides server implementations for external providers.
package server

import (
	"context"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/crossplane/crossplane-runtime/apis/proto/external/v1alpha1"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

// Error strings.
const (
	errMarshalManagedResource           = "cannot marshal managed resource to JSON"
	errUnmarshalManagedResource         = "cannot unmarshal managed resource from JSON"
	errStreamSend                       = "failed to send response on stream"
	errStreamRecv                       = "failed to receive request from stream"
	errNoMatchingOperation              = "no matching operation found in request"
	errNoMatchingResourceTypeHandler    = "no matching resource type handler found"
	errRegisterNilHandler               = "cannot register nil handler"
	errRegisterEmptyGroupVersionKind    = "cannot register handler for empty GroupVersionKind"
	errRegisterHandlerAlreadyRegistered = "handler for GroupVersionKind is already registered"
	errManagedResourceConversion        = "failed to convert to typed managed resource"
	errCannotDetermineResourceType      = "cannot determine resource type from struct"
)

// A TypeHandlerMap maps GroupVersionKind to TypedExternalConnector instances.
type TypeHandlerMap map[schema.GroupVersionKind]managed.TypedExternalConnecter[resource.Managed]

// A ProviderServer implements the ExternalService gRPC service.
type ProviderServer struct {
	v1alpha1.UnimplementedExternalServiceServer

	// handlers maps resource types to their respective handlers.
	handlers TypeHandlerMap

	// scheme is used for managed resource encoding/decoding.
	scheme *runtime.Scheme

	// mu protects the handlers map.
	mu sync.RWMutex

	// log is the server logger.
	log logging.Logger

	// opts contains gRPC server options.
	opts []grpc.ServerOption
}

// A ProviderServerOption configures a ProviderServer.
type ProviderServerOption func(*ProviderServer)

// WithServerLogger sets the logger for the ProviderServer.
func WithServerLogger(log logging.Logger) ProviderServerOption {
	return func(s *ProviderServer) {
		s.log = log
	}
}

// WithTLSCredentials configures the server to use TLS credentials.
func WithTLSCredentials(creds credentials.TransportCredentials) ProviderServerOption {
	return func(s *ProviderServer) {
		s.opts = append(s.opts, grpc.Creds(creds))
	}
}

// WithGRPCServerOptions configures additional gRPC server options.
func WithGRPCServerOptions(opts ...grpc.ServerOption) ProviderServerOption {
	return func(s *ProviderServer) {
		s.opts = append(s.opts, opts...)
	}
}

// NewProviderServer creates a new ProviderServer with the given options.
func NewProviderServer(scheme *runtime.Scheme, o ...ProviderServerOption) *ProviderServer {
	s := &ProviderServer{
		handlers: make(TypeHandlerMap),
		scheme:   scheme,
		log:      logging.NewNopLogger(),
	}

	for _, opt := range o {
		opt(s)
	}

	return s
}

// RegisterHandler registers a TypedExternalConnecter for a specific managed resource type.
func (s *ProviderServer) RegisterHandler(gvk schema.GroupVersionKind, h managed.TypedExternalConnecter[resource.Managed]) error {
	if h == nil {
		return errors.New(errRegisterNilHandler)
	}

	if gvk.Empty() {
		return errors.New(errRegisterEmptyGroupVersionKind)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.handlers[gvk]; exists {
		return errors.Errorf(errRegisterHandlerAlreadyRegistered, gvk.String())
	}

	s.handlers[gvk] = h
	s.log.Debug("Registered handler", "gvk", gvk.String())
	return nil
}

// Discover returns information about the available resource types handled by this server.
func (s *ProviderServer) Discover(_ context.Context, _ *v1alpha1.DiscoveryRequest) (*v1alpha1.DiscoveryResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resp := &v1alpha1.DiscoveryResponse{
		Version: "v1alpha1", // TODO: Make this configurable or derive from build info
	}

	for gvk := range s.handlers {
		rec := &v1alpha1.Reconciler{
			Named: gvk.String(),
			For: &v1alpha1.Reconciler_Reference{
				ApiVersion: gvk.GroupVersion().String(),
				Kind:       gvk.Kind,
			},
		}
		resp.Reconcilers = append(resp.Reconcilers, rec)
	}

	return resp, nil
}

// RegisterWithServer registers this provider server with the given gRPC server.
func (s *ProviderServer) RegisterWithServer(server *grpc.Server) {
	v1alpha1.RegisterExternalServiceServer(server, s)
}

// Serve starts the gRPC server on the specified address.
func (s *ProviderServer) Serve(address string) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	server := grpc.NewServer(s.opts...)
	s.RegisterWithServer(server)

	s.log.Info("Starting gRPC server", "address", address)
	return server.Serve(lis)
}

// ServerFactory returns a factory function that creates a new ProviderServer.
// This allows the creation of the server to be deferred until it's needed.
type ServerFactory func() (*ProviderServer, error)
