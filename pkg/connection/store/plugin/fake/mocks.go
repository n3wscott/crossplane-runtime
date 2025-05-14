/*
 Copyright 2023 The Crossplane Authors.
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

// Package fake is a fake ExternalSecretStorePluginServiceClient.
package fake

import (
	"context"
	"github.com/crossplane/crossplane-runtime/apis/proto/ess/v1alpha1"

	"google.golang.org/grpc"
)

// ExternalSecretStorePluginServiceClient is a fake ExternalSecretStorePluginServiceClient.
type ExternalSecretStorePluginServiceClient struct {
	GetSecretFn   func(context.Context, *v1alpha1.GetSecretRequest, ...grpc.CallOption) (*v1alpha1.GetSecretResponse, error)
	ApplySecretFn func(context.Context, *v1alpha1.ApplySecretRequest, ...grpc.CallOption) (*v1alpha1.ApplySecretResponse, error)
	DeleteKeysFn  func(context.Context, *v1alpha1.DeleteKeysRequest, ...grpc.CallOption) (*v1alpha1.DeleteKeysResponse, error)
	*v1alpha1.UnimplementedExternalSecretStorePluginServiceServer
}

// GetSecret returns the secret.
func (e *ExternalSecretStorePluginServiceClient) GetSecret(ctx context.Context, req *v1alpha1.GetSecretRequest, _ ...grpc.CallOption) (*v1alpha1.GetSecretResponse, error) {
	return e.GetSecretFn(ctx, req)
}

// ApplySecret applies the secret.
func (e *ExternalSecretStorePluginServiceClient) ApplySecret(ctx context.Context, req *v1alpha1.ApplySecretRequest, _ ...grpc.CallOption) (*v1alpha1.ApplySecretResponse, error) {
	return e.ApplySecretFn(ctx, req)
}

// DeleteKeys deletes the secret keys.
func (e *ExternalSecretStorePluginServiceClient) DeleteKeys(ctx context.Context, req *v1alpha1.DeleteKeysRequest, _ ...grpc.CallOption) (*v1alpha1.DeleteKeysResponse, error) {
	return e.DeleteKeysFn(ctx, req)
}
