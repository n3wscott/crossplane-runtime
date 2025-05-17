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

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

// fromProtoStruct converts a protobuf Struct to a managed resource.
func fromProtoStruct(scheme *runtime.Scheme, pb *structpb.Struct) (resource.Managed, schema.GroupVersionKind, error) {
	if pb == nil {
		return nil, schema.GroupVersionKind{}, errors.New("nil resource struct")
	}

	// Extract the GVK from the struct (assuming it's in the typical metadata.apiVersion and kind)
	var gvk schema.GroupVersionKind
	if apiVersionField, ok := pb.Fields["apiVersion"]; ok && apiVersionField.GetStringValue() != "" {
		gv, err := schema.ParseGroupVersion(apiVersionField.GetStringValue())
		if err != nil {
			return nil, schema.GroupVersionKind{}, err
		}
		if kindField, ok := pb.Fields["kind"]; ok && kindField.GetStringValue() != "" {
			gvk = gv.WithKind(kindField.GetStringValue())
		}
	}

	if gvk.Empty() {
		return nil, schema.GroupVersionKind{}, errors.New(errCannotDetermineResourceType)
	}

	// Convert the protobuf struct to JSON
	b, err := protojson.Marshal(pb)
	if err != nil {
		return nil, gvk, errors.Wrap(err, errMarshalManagedResource)
	}

	// Create a new instance of the managed resource
	obj, err := scheme.New(gvk)
	if err != nil {
		return nil, gvk, errors.Wrap(err, "cannot create new object of type "+gvk.String())
	}

	// Unmarshal the JSON into the managed resource
	if err := json.Unmarshal(b, obj); err != nil {
		return nil, gvk, errors.Wrap(err, errUnmarshalManagedResource)
	}

	// Convert to resource.Managed
	mg, ok := obj.(resource.Managed)
	if !ok {
		return nil, gvk, errors.Errorf("%s is not a managed resource", gvk.String())
	}

	return mg, gvk, nil
}