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
	"github.com/crossplane/crossplane-runtime/apis/proto/external/v1alpha1"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
)

// ToObserveResponse converts an ExternalObservation to an ObserveResponse.
func ToObserveResponse(obs managed.ExternalObservation, resourceStruct interface{}) *v1alpha1.ObserveResponse {
	s, _ := AsStruct(resourceStruct)
	return &v1alpha1.ObserveResponse{
		Resource:                s,
		ConnectionDetails:       obs.ConnectionDetails,
		ResourceExists:          obs.ResourceExists,
		ResourceUpToDate:        obs.ResourceUpToDate,
		ResourceLateInitialized: obs.ResourceLateInitialized,
	}
}

// ToCreateResponse converts an ExternalCreation to a CreateResponse.
func ToCreateResponse(creation managed.ExternalCreation, resourceStruct interface{}) *v1alpha1.CreateResponse {
	s, _ := AsStruct(resourceStruct)
	return &v1alpha1.CreateResponse{
		Resource:          s,
		ConnectionDetails: creation.ConnectionDetails,
		AdditionalDetails: creation.AdditionalDetails,
	}
}

// ToUpdateResponse converts an ExternalUpdate to an UpdateResponse.
func ToUpdateResponse(update managed.ExternalUpdate, resourceStruct interface{}) *v1alpha1.UpdateResponse {
	s, _ := AsStruct(resourceStruct)
	return &v1alpha1.UpdateResponse{
		Resource:          s,
		ConnectionDetails: update.ConnectionDetails,
		AdditionalDetails: update.AdditionalDetails,
	}
}

// ToDeleteResponse converts an ExternalDelete to a DeleteResponse.
func ToDeleteResponse(deletion managed.ExternalDelete, resourceStruct interface{}) *v1alpha1.DeleteResponse {
	s, _ := AsStruct(resourceStruct)
	return &v1alpha1.DeleteResponse{
		Resource:          s,
		AdditionalDetails: deletion.AdditionalDetails,
	}
}