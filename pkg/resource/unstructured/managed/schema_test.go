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
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestSchemeRegistration(t *testing.T) {
	// Define test GroupVersionKind
	gvk := schema.GroupVersionKind{
		Group:   "test.crossplane.io",
		Version: "v1alpha1",
		Kind:    "TestResource",
	}

	// Create a managed.Unstructured with the GVK
	orig := New(WithGroupVersionKind(gvk))

	// Setup a new runtime scheme
	scheme := runtime.NewScheme()

	// Register our Unstructured type with the specific GVK
	scheme.AddKnownTypeWithName(gvk, orig)

	// Register our defaulting function
	scheme.AddTypeDefaultingFunc(orig, func(obj interface{}) {
		if u, ok := obj.(*Unstructured); ok {
			u.Default()
		}
	})

	// Test getting the type from the scheme
	obj, err := scheme.New(gvk)
	if err != nil {
		t.Fatalf("Failed to create object from scheme: %v", err)
	}

	// Convert to our managed type
	got, ok := obj.(*Unstructured)
	if !ok {
		t.Fatalf("Expected *managed.Unstructured, got %T", obj)
	}

	// Apply the defaulting function (this would be done by the API server in practice)
	scheme.Default(got)

	// Verify the Object map is initialized
	if got.Object == nil {
		t.Fatal("Object map is nil, should be initialized by Default function")
	}

	// Verify the GVK is preserved when creating a new instance
	got.SetGroupVersionKind(gvk)

	if diff := cmp.Diff(orig.GetObjectKind().GroupVersionKind(), got.GetObjectKind().GroupVersionKind()); diff != "" {
		t.Errorf("GroupVersionKind differs: -want, +got:\n%s", diff)
	}
}

func TestSchemeRegistrationWithDeepCopy(t *testing.T) {
	// Define test GroupVersionKind
	gvk := schema.GroupVersionKind{
		Group:   "test.crossplane.io",
		Version: "v1alpha1",
		Kind:    "TestResource",
	}

	// Create a managed.Unstructured with the GVK
	orig := New(WithGroupVersionKind(gvk))

	// Create a deep copy of the object
	copiedObj := orig.DeepCopyObject()

	// Verify the copy is of the right type
	copied, ok := copiedObj.(*Unstructured)
	if !ok {
		t.Fatalf("Expected *managed.Unstructured, got %T", copiedObj)
	}

	// Verify the GVK is preserved in the copy
	if diff := cmp.Diff(orig.GetObjectKind().GroupVersionKind(), copied.GetObjectKind().GroupVersionKind()); diff != "" {
		t.Errorf("GroupVersionKind differs after DeepCopy: -want, +got:\n%s", diff)
	}
}

func TestObjectRehydration(t *testing.T) {
	// Define test GroupVersionKind
	gvk := schema.GroupVersionKind{
		Group:   "test.crossplane.io",
		Version: "v1alpha1",
		Kind:    "TestResource",
	}

	// Create a managed.Unstructured with the GVK
	reg := New(WithGroupVersionKind(gvk))

	// Create a scheme and register our type
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(gvk, reg)

	// Register our defaulting function
	scheme.AddTypeDefaultingFunc(reg, func(obj interface{}) {
		if u, ok := obj.(*Unstructured); ok {
			u.Default()
			// Make sure to explicitly set the GVK since it's not automatically preserved
			u.SetGroupVersionKind(gvk)
		}
	})

	// First, create a managed.Unstructured with the GVK and some data
	orig := New(WithGroupVersionKind(gvk))
	orig.Object["spec"] = map[string]interface{}{
		"foo": "bar",
	}

	// Convert to runtime.Object
	var runtimeObj runtime.Object = orig

	// Now ask the scheme to create a new instance of our type
	newObj, err := scheme.New(runtimeObj.GetObjectKind().GroupVersionKind())
	if err != nil {
		t.Fatalf("Failed to create object from scheme: %v", err)
	}

	// Apply the defaulting function (this would be done by the API server in practice)
	scheme.Default(newObj)

	// The new object should have the same GVK but empty content
	newManaged, ok := newObj.(*Unstructured)
	if !ok {
		t.Fatalf("Expected *managed.Unstructured, got %T", newObj)
	}

	// Verify the new object has the correct GVK
	if diff := cmp.Diff(orig.GetObjectKind().GroupVersionKind(), newManaged.GetObjectKind().GroupVersionKind()); diff != "" {
		t.Errorf("GroupVersionKind differs for new object: -want, +got:\n%s", diff)
	}

	// The new managed object should have its Object map initialized
	if newManaged.Object == nil {
		t.Error("New Unstructured instance has nil Object map")
	}
}

func TestEmptyObjectInitialization(t *testing.T) {
	// This tests that when the scheme creates a new instance,
	// the Object map is properly initialized

	// Define test GroupVersionKind
	gvk := schema.GroupVersionKind{
		Group:   "test.crossplane.io",
		Version: "v1alpha1",
		Kind:    "TestResource",
	}

	orig := New(WithGroupVersionKind(gvk))

	// Create a scheme and register our type
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(gvk, orig)

	// Register our defaulting function
	scheme.AddTypeDefaultingFunc(orig, func(obj interface{}) {
		if u, ok := obj.(*Unstructured); ok {
			u.Default()
		}
	})

	// Create a new instance through the scheme
	obj, err := scheme.New(gvk)
	if err != nil {
		t.Fatalf("Failed to create object from scheme: %v", err)
	}

	// Convert to our managed type
	managed, ok := obj.(*Unstructured)
	if !ok {
		t.Fatalf("Expected *managed.Unstructured, got %T", obj)
	}

	// Apply the defaulting function
	scheme.Default(managed)

	// Verify the object map is initialized after defaulting
	if managed.Object == nil {
		t.Error("Object map is nil, should be initialized by Default function")
	}

	// Set GVK and verify it works (should not panic)
	managed.SetGroupVersionKind(gvk)

	// Verify GVK was set correctly
	gotGVK := managed.GetObjectKind().GroupVersionKind()
	if gotGVK != gvk {
		t.Errorf("Expected GVK %v, got %v", gvk, gotGVK)
	}
}

func TestDefault(t *testing.T) {
	cases := map[string]struct {
		u *Unstructured
	}{
		"NilObject": {
			u: &Unstructured{},
		},
		"EmptyObject": {
			u: &Unstructured{Unstructured: unstructured.Unstructured{Object: map[string]interface{}{}}},
		},
		"PopulatedObject": {
			u: &Unstructured{Unstructured: unstructured.Unstructured{Object: map[string]interface{}{
				"foo": "bar",
			}}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			wasNil := tc.u.Object == nil

			// Call Default
			tc.u.Default()

			// Verify Object is initialized
			if tc.u.Object == nil {
				t.Fatal("Object map is still nil after calling Default")
			}

			// For cases that start with a non-nil Object map, verify we didn't change the map
			if !wasNil {
				if _, has := tc.u.Object["foo"]; name == "PopulatedObject" && !has {
					t.Error("Existing Object map data was lost")
				}
			}
		})
	}
}

func TestDefaultPreservesGVK(t *testing.T) {
	// Define test GroupVersionKind
	gvk := schema.GroupVersionKind{
		Group:   "test.crossplane.io",
		Version: "v1alpha1",
		Kind:    "TestResource",
	}

	// Create an Unstructured with no Object map but with a GVK set
	u := &Unstructured{}
	u.SetGroupVersionKind(gvk)

	// Call Default
	u.Default()

	// Verify the GVK is still set
	if u.GetObjectKind().GroupVersionKind() != gvk {
		t.Errorf("GVK was lost after calling Default, expected %v, got %v",
			gvk, u.GetObjectKind().GroupVersionKind())
	}

	// Verify apiVersion and kind are set in the Object map
	apiVersion, found, err := unstructured.NestedString(u.Object, "apiVersion")
	if err != nil || !found || apiVersion != gvk.GroupVersion().String() {
		t.Errorf("apiVersion not properly set in Object map after Default, expected %q, got %q",
			gvk.GroupVersion().String(), apiVersion)
	}

	kind, found, err := unstructured.NestedString(u.Object, "kind")
	if err != nil || !found || kind != gvk.Kind {
		t.Errorf("kind not properly set in Object map after Default, expected %q, got %q",
			gvk.Kind, kind)
	}
}
