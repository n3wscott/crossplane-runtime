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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

// TypeTag is an empty interface used to create unique types
// for TypedUnstructured.
type TypeTag interface{}

// TypedUnstructured is an unstructured managed resource with a unique type
// based on its type parameter T, which allows multiple instances to be registered
// with different GVKs in the same scheme.
type TypedUnstructured[T TypeTag] struct {
	// Embed the regular Unstructured type to inherit all methods
	Unstructured
}

// NewTyped creates a new TypedUnstructured with the given TypeTag and options.
func NewTyped[T TypeTag](gvk schema.GroupVersionKind, opts ...Option) *TypedUnstructured[T] {
	u := &TypedUnstructured[T]{}
	u.Default()
	u.SetGroupVersionKind(gvk)
	for _, f := range opts {
		f(&u.Unstructured)
	}
	return u
}

// Default ensures the Object map is initialized.
func (u *TypedUnstructured[T]) Default() {
	u.Unstructured.Default()
}

// DeepCopyObject returns a deep copy of the TypedUnstructured.
func (u *TypedUnstructured[T]) DeepCopyObject() runtime.Object {
	if u == nil {
		return nil
	}
	out := &TypedUnstructured[T]{}
	if u.Object != nil {
		out.Object = runtime.DeepCopyJSON(u.Object)
	}
	return out
}

// GetCondition from this TypedUnstructured resource.
func (u *TypedUnstructured[T]) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return u.Unstructured.GetCondition(ct)
}

// SetConditions for this TypedUnstructured resource.
func (u *TypedUnstructured[T]) SetConditions(c ...xpv1.Condition) {
	u.Unstructured.SetConditions(c...)
}

// GetProviderConfigReference of this TypedUnstructured resource.
func (u *TypedUnstructured[T]) GetProviderConfigReference() *xpv1.Reference {
	return u.Unstructured.GetProviderConfigReference()
}

// SetProviderConfigReference of this TypedUnstructured resource.
func (u *TypedUnstructured[T]) SetProviderConfigReference(p *xpv1.Reference) {
	u.Unstructured.SetProviderConfigReference(p)
}

// GetWriteConnectionSecretToReference of this TypedUnstructured resource.
func (u *TypedUnstructured[T]) GetWriteConnectionSecretToReference() *xpv1.SecretReference {
	return u.Unstructured.GetWriteConnectionSecretToReference()
}

// SetWriteConnectionSecretToReference of this TypedUnstructured resource.
func (u *TypedUnstructured[T]) SetWriteConnectionSecretToReference(r *xpv1.SecretReference) {
	u.Unstructured.SetWriteConnectionSecretToReference(r)
}

// GetPublishConnectionDetailsTo of this TypedUnstructured resource.
func (u *TypedUnstructured[T]) GetPublishConnectionDetailsTo() *xpv1.PublishConnectionDetailsTo {
	return u.Unstructured.GetPublishConnectionDetailsTo()
}

// SetPublishConnectionDetailsTo of this TypedUnstructured resource.
func (u *TypedUnstructured[T]) SetPublishConnectionDetailsTo(ref *xpv1.PublishConnectionDetailsTo) {
	u.Unstructured.SetPublishConnectionDetailsTo(ref)
}

// GetManagementPolicies of this TypedUnstructured resource.
func (u *TypedUnstructured[T]) GetManagementPolicies() xpv1.ManagementPolicies {
	return u.Unstructured.GetManagementPolicies()
}

// SetManagementPolicies of this TypedUnstructured resource.
func (u *TypedUnstructured[T]) SetManagementPolicies(p xpv1.ManagementPolicies) {
	u.Unstructured.SetManagementPolicies(p)
}

// GetDeletionPolicy of this TypedUnstructured resource.
func (u *TypedUnstructured[T]) GetDeletionPolicy() xpv1.DeletionPolicy {
	return u.Unstructured.GetDeletionPolicy()
}

// SetDeletionPolicy of this TypedUnstructured resource.
func (u *TypedUnstructured[T]) SetDeletionPolicy(p xpv1.DeletionPolicy) {
	u.Unstructured.SetDeletionPolicy(p)
}

// SetObservedGeneration of this TypedUnstructured resource.
func (u *TypedUnstructured[T]) SetObservedGeneration(generation int64) {
	u.Unstructured.SetObservedGeneration(generation)
}

// GetObservedGeneration of this TypedUnstructured resource.
func (u *TypedUnstructured[T]) GetObservedGeneration() int64 {
	return u.Unstructured.GetObservedGeneration()
}

// GetUnstructured returns the underlying *unstructured.Unstructured.
func (u *TypedUnstructured[T]) GetUnstructured() *unstructured.Unstructured {
	return &u.Unstructured.Unstructured
}

// Ensure we implement the resource.Managed interface
var _ resource.Managed = &TypedUnstructured[struct{}]{}
