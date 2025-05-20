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

package test

import (
	"context"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

// MockConnectFn mocks the Connect method of an ExternalConnector.
type MockConnectFn func(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error)

// MockConnector is a mock ExternalConnector for testing.
type MockConnector struct {
	MockConnect MockConnectFn
}

// Connect calls the MockConnect function.
func (c *MockConnector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	if c.MockConnect != nil {
		return c.MockConnect(ctx, mg)
	}
	return &MockExternalClient{}, nil
}

// NewMockConnector creates a new MockConnector.
func NewMockConnector() *MockConnector {
	return &MockConnector{}
}

// WithMockConnect adds a MockConnect function to the MockConnector.
func WithMockConnect(fn MockConnectFn) func(*MockConnector) {
	return func(c *MockConnector) {
		c.MockConnect = fn
	}
}

// MockObserveFn mocks the Observe method of an ExternalClient.
type MockObserveFn func(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error)

// MockExternalCreateFn mocks the Create method of an ExternalClient.
type MockExternalCreateFn func(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error)

// MockExternalUpdateFn mocks the Update method of an ExternalClient.
type MockExternalUpdateFn func(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error)

// MockExternalDeleteFn mocks the Delete method of an ExternalClient.
type MockExternalDeleteFn func(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error)

// MockDisconnectFn mocks the Disconnect method of an ExternalClient.
type MockDisconnectFn func(ctx context.Context) error

// MockExternalClient is a mock ExternalClient for testing.
type MockExternalClient struct {
	MockObserve    MockObserveFn
	MockCreate     MockExternalCreateFn
	MockUpdate     MockExternalUpdateFn
	MockDelete     MockExternalDeleteFn
	MockDisconnect MockDisconnectFn
}

// Observe calls the MockObserve function.
func (c *MockExternalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	if c.MockObserve != nil {
		return c.MockObserve(ctx, mg)
	}
	return managed.ExternalObservation{ResourceExists: false}, nil
}

// Create calls the MockCreate function.
func (c *MockExternalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	if c.MockCreate != nil {
		return c.MockCreate(ctx, mg)
	}
	return managed.ExternalCreation{}, nil
}

// Update calls the MockUpdate function.
func (c *MockExternalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	if c.MockUpdate != nil {
		return c.MockUpdate(ctx, mg)
	}
	return managed.ExternalUpdate{}, nil
}

// Delete calls the MockDelete function.
func (c *MockExternalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	if c.MockDelete != nil {
		return c.MockDelete(ctx, mg)
	}
	return managed.ExternalDelete{}, nil
}

// WithMockObserve adds a MockObserve function to the MockExternalClient.
func WithMockObserve(fn MockObserveFn) func(*MockExternalClient) {
	return func(c *MockExternalClient) {
		c.MockObserve = fn
	}
}

// WithMockCreate adds a MockCreate function to the MockExternalClient.
func WithMockCreate(fn MockExternalCreateFn) func(*MockExternalClient) {
	return func(c *MockExternalClient) {
		c.MockCreate = fn
	}
}

// WithMockUpdate adds a MockUpdate function to the MockExternalClient.
func WithMockUpdate(fn MockExternalUpdateFn) func(*MockExternalClient) {
	return func(c *MockExternalClient) {
		c.MockUpdate = fn
	}
}

// WithMockDelete adds a MockDelete function to the MockExternalClient.
func WithMockDelete(fn MockExternalDeleteFn) func(*MockExternalClient) {
	return func(c *MockExternalClient) {
		c.MockDelete = fn
	}
}

// Disconnect calls the MockDisconnect function.
func (c *MockExternalClient) Disconnect(ctx context.Context) error {
	if c.MockDisconnect != nil {
		return c.MockDisconnect(ctx)
	}
	return nil
}

// WithMockDisconnect adds a MockDisconnect function to the MockExternalClient.
func WithMockDisconnect(fn MockDisconnectFn) func(*MockExternalClient) {
	return func(c *MockExternalClient) {
		c.MockDisconnect = fn
	}
}