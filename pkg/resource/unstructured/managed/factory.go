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

package managed

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TypeTagFactory creates managed resources based on GVK information
type TypeTagFactory struct {
	// Map of GVK to creator function
	creators map[schema.GroupVersionKind]func() runtime.Object
}

// NewTypeTagFactory creates a new TypeTagFactory.
func NewTypeTagFactory() *TypeTagFactory {
	return &TypeTagFactory{
		creators: make(map[schema.GroupVersionKind]func() runtime.Object),
	}
}

// RegisterType registers a GVK with the factory.
// For compatibility with existing code, this returns nil.
func (f *TypeTagFactory) RegisterType(gvk schema.GroupVersionKind) interface{} {
	// Register a creator function for this GVK
	f.creators[gvk] = func() runtime.Object {
		// Create a regular unstructured object
		obj := &Unstructured{}
		obj.Default()
		obj.SetGroupVersionKind(gvk)
		return obj
	}
	
	return nil
}

// CreateObject creates a new instance of a managed resource for the given GVK.
func (f *TypeTagFactory) CreateObject(gvk schema.GroupVersionKind) runtime.Object {
	// Get the creator function for this GVK
	creator, found := f.creators[gvk]
	if !found {
		// Register first
		f.RegisterType(gvk)
		creator = f.creators[gvk]
	}

	return creator()
}

// RegisterWithScheme registers the managed type with the given scheme for a specific GVK.
func (f *TypeTagFactory) RegisterWithScheme(scheme *runtime.Scheme, gvk schema.GroupVersionKind) error {
	// Create a sample object 
	obj := f.CreateObject(gvk)

	// Register it with the scheme
	scheme.AddKnownTypeWithName(gvk, obj)

	// Register a defaulting function
	scheme.AddTypeDefaultingFunc(obj, func(o interface{}) {
		if u, ok := o.(*Unstructured); ok {
			u.Default()
		}
	})

	return nil
}
