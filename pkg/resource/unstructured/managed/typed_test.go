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
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

// Ensure TypedUnstructured implements resource.Managed
var _ resource.Managed = &TypedUnstructured[struct{}]{}

// Define tag types to get different reflect.Type
type Tag1 struct{}
type Tag2 struct{}
type Tag3 struct{}

func TestTypedUnstructured(t *testing.T) {
	// Define test GVKs
	gvk1 := schema.GroupVersionKind{
		Group:   "test.crossplane.io",
		Version: "v1alpha1",
		Kind:    "TestResource1",
	}
	gvk2 := schema.GroupVersionKind{
		Group:   "test.crossplane.io",
		Version: "v1alpha1",
		Kind:    "TestResource2",
	}

	// Create typed unstructured objects
	u1 := NewTyped[Tag1](gvk1)
	u2 := NewTyped[Tag2](gvk2)

	// Verify that they're both properly initialized and have correct GVKs
	if u1.Object == nil {
		t.Error("Object map is nil for u1")
	}
	if u2.Object == nil {
		t.Error("Object map is nil for u2")
	}

	if got := u1.GetObjectKind().GroupVersionKind(); got != gvk1 {
		t.Errorf("Expected GVK %v, got %v for u1", gvk1, got)
	}
	if got := u2.GetObjectKind().GroupVersionKind(); got != gvk2 {
		t.Errorf("Expected GVK %v, got %v for u2", gvk2, got)
	}

	// Verify that they're different types
	if reflect.TypeOf(u1) == reflect.TypeOf(u2) {
		t.Error("Expected different types for u1 and u2")
	}
}

func TestTypedUnstructuredSchemeRegistration(t *testing.T) {
	// Define test GVKs
	gvk1 := schema.GroupVersionKind{
		Group:   "test.crossplane.io",
		Version: "v1alpha1",
		Kind:    "TestResource1",
	}
	gvk2 := schema.GroupVersionKind{
		Group:   "test.crossplane.io",
		Version: "v1alpha1",
		Kind:    "TestResource2",
	}

	// Create typed objects with different tags
	u1 := NewTyped[Tag1](gvk1)
	u2 := NewTyped[Tag2](gvk2)

	// Setup a new runtime scheme
	scheme := runtime.NewScheme()

	// Register both types with their GVKs
	scheme.AddKnownTypeWithName(gvk1, u1.DeepCopyObject())
	scheme.AddKnownTypeWithName(gvk2, u2.DeepCopyObject())

	// Register defaulting functions
	scheme.AddTypeDefaultingFunc(u1, func(obj interface{}) {
		if u, ok := obj.(*TypedUnstructured[Tag1]); ok {
			u.Default()
		}
	})
	scheme.AddTypeDefaultingFunc(u2, func(obj interface{}) {
		if u, ok := obj.(*TypedUnstructured[Tag2]); ok {
			u.Default()
		}
	})

	// Test creating objects from the scheme
	obj1, err := scheme.New(gvk1)
	if err != nil {
		t.Fatalf("Failed to create object from scheme for gvk1: %v", err)
	}
	obj2, err := scheme.New(gvk2)
	if err != nil {
		t.Fatalf("Failed to create object from scheme for gvk2: %v", err)
	}

	// Verify objects are of the correct types
	got1, ok := obj1.(*TypedUnstructured[Tag1])
	if !ok {
		t.Fatalf("Expected *managed.TypedUnstructured[Tag1], got %T", obj1)
	}
	got2, ok := obj2.(*TypedUnstructured[Tag2])
	if !ok {
		t.Fatalf("Expected *managed.TypedUnstructured[Tag2], got %T", obj2)
	}

	// Apply defaulting
	scheme.Default(got1)
	scheme.Default(got2)

	// Verify the Object maps are initialized
	if got1.Object == nil {
		t.Error("Object map is nil for got1")
	}
	if got2.Object == nil {
		t.Error("Object map is nil for got2")
	}

	// Set the GVKs and verify they're retained
	got1.SetGroupVersionKind(gvk1)
	got2.SetGroupVersionKind(gvk2)

	if diff := cmp.Diff(u1.GetObjectKind().GroupVersionKind(), got1.GetObjectKind().GroupVersionKind()); diff != "" {
		t.Errorf("GroupVersionKind differs for gvk1: -want, +got:\n%s", diff)
	}
	if diff := cmp.Diff(u2.GetObjectKind().GroupVersionKind(), got2.GetObjectKind().GroupVersionKind()); diff != "" {
		t.Errorf("GroupVersionKind differs for gvk2: -want, +got:\n%s", diff)
	}
}

func TestSimpleObjectCreation(t *testing.T) {
	// Define test GVKs
	gvk1 := schema.GroupVersionKind{
		Group:   "test.crossplane.io",
		Version: "v1alpha1",
		Kind:    "TestResource1",
	}
	gvk2 := schema.GroupVersionKind{
		Group:   "test.crossplane.io",
		Version: "v1alpha1",
		Kind:    "TestResource2",
	}

	// Create typed objects directly
	obj1 := NewTyped[Tag1](gvk1)
	obj2 := NewTyped[Tag2](gvk2)

	// Verify the objects have the correct GVKs
	if obj1.GetObjectKind().GroupVersionKind() != gvk1 {
		t.Errorf("Object 1 has incorrect GVK: %v", obj1.GetObjectKind().GroupVersionKind())
	}
	if obj2.GetObjectKind().GroupVersionKind() != gvk2 {
		t.Errorf("Object 2 has incorrect GVK: %v", obj2.GetObjectKind().GroupVersionKind())
	}
	
	// Verify the objects are properly initialized
	if obj1.Object == nil {
		t.Error("Object 1 has nil Object map")
	}
	if obj2.Object == nil {
		t.Error("Object 2 has nil Object map")
	}
}
