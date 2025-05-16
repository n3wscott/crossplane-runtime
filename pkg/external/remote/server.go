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

	"github.com/crossplane/crossplane-runtime/apis/proto/external/v1alpha1"
)

var _ v1alpha1.ExternalServiceServer = &StreamingServer{}

type StreamingServer struct {
	v1alpha1.UnimplementedExternalServiceServer
}

func (s *StreamingServer) Discover(ctx context.Context, request *v1alpha1.DiscoveryRequest) (*v1alpha1.DiscoveryResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s *StreamingServer) Session(server v1alpha1.ExternalService_SessionServer) error {
	//TODO implement me
	panic("implement me")
}

var _ v1alpha1.ConnectedExternalServiceServer = &Server{}

type Server struct {
	v1alpha1.UnimplementedConnectedExternalServiceServer
}

func (s Server) Observe(ctx context.Context, request *v1alpha1.ObserveRequest) (*v1alpha1.ObserveResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) Create(ctx context.Context, request *v1alpha1.CreateRequest) (*v1alpha1.CreateResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) Update(ctx context.Context, request *v1alpha1.UpdateRequest) (*v1alpha1.UpdateResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) Delete(ctx context.Context, request *v1alpha1.DeleteRequest) (*v1alpha1.DeleteResponse, error) {
	//TODO implement me
	panic("implement me")
}
