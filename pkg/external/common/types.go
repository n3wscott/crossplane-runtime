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

package common

import (
	"context"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/crossplane/crossplane-runtime/apis/proto/external/v1alpha1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

// Logger returns the logger for this component.
type Logger interface {
	// WithValues returns a logger with additional key-value pairs.
	WithValues(keysAndValues ...any) logging.Logger
}

// GRPCEndpointOptions holds configuration options for gRPC endpoints.
type GRPCEndpointOptions struct {
	// Endpoint is the gRPC server address.
	Endpoint string

	// UseTLS indicates whether to use TLS for the connection.
	UseTLS bool

	// ResourceType is the type of resource being handled.
	ResourceType schema.GroupVersionKind
}

// ConnectionManager manages connections to external systems.
type ConnectionManager interface {
	// GetConnection returns an existing connection for the given ID, or nil if none exists.
	GetConnection(id string) managed.ExternalClient
	
	// AddConnection adds a new connection with the given ID.
	AddConnection(id string, client managed.ExternalClient)
	
	// RemoveConnection removes a connection with the given ID.
	RemoveConnection(id string)
}

// ResourceHandler handles resources of a specific type.
type ResourceHandler interface {
	// GetGVK returns the GroupVersionKind this handler can process.
	GetGVK() schema.GroupVersionKind
	
	// Connect establishes a connection to the external system.
	Connect(mg resource.Managed) (managed.ExternalClient, error)
}

// HandlerInfo contains information about a registered resource handler.
type HandlerInfo struct {
	// GVK is the GroupVersionKind of the resource.
	GVK schema.GroupVersionKind
	
	// Handler is the connector for this resource type.
	Handler managed.TypedExternalConnecter[resource.Managed]
}

// SessionManager defines the interface for managing gRPC sessions.
type SessionManager interface {
	// OpenSession starts a new bidirectional session with the server.
	OpenSession(ctx context.Context) (v1alpha1.ExternalService_SessionClient, error)
	
	// CloseSession closes an open session.
	CloseSession(client v1alpha1.ExternalService_SessionClient) error
}