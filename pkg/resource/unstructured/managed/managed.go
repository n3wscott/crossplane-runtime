/*
Copyright 2020 The Crossplane Authors.

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

// Package managed contains an unstructured managed resource.
package managed

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/fieldpath"
)

// An Option modifies an unstructured managed resource.
type Option func(resource *Unstructured)

// WithGroupVersionKind sets the GroupVersionKind of the unstructured managed
// resource.
func WithGroupVersionKind(gvk schema.GroupVersionKind) Option {
	return func(c *Unstructured) {
		c.SetGroupVersionKind(gvk)
	}
}

// WithConditions returns an Option that sets the supplied conditions on an
// unstructured managed resource.
func WithConditions(c ...xpv1.Condition) Option {
	return func(cr *Unstructured) {
		cr.SetConditions(c...)
	}
}

// New returns a new unstructured managed resource.
func New(opts ...Option) *Unstructured {
	cr := &Unstructured{unstructured.Unstructured{Object: make(map[string]any)}}
	for _, f := range opts {
		f(cr)
	}
	return cr
}

// +k8s:deepcopy-gen=true
// +kubebuilder:object:root=true

// An Unstructured managed resource.
type Unstructured struct {
	unstructured.Unstructured
}

// GetUnstructured returns the underlying *unstructured.Unstructured.
func (cr *Unstructured) GetUnstructured() *unstructured.Unstructured {
	return &cr.Unstructured
}

// GetCondition of this Managed resource.
func (cr *Unstructured) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	conditioned := xpv1.ConditionedStatus{}
	// The path is directly `status` because conditions are inline.
	if err := fieldpath.Pave(cr.Object).GetValueInto("status", &conditioned); err != nil {
		return xpv1.Condition{}
	}
	return conditioned.GetCondition(ct)
}

// SetConditions of this Managed resource.
func (cr *Unstructured) SetConditions(c ...xpv1.Condition) {
	conditioned := xpv1.ConditionedStatus{}
	// The path is directly `status` because conditions are inline.
	_ = fieldpath.Pave(cr.Object).GetValueInto("status", &conditioned)
	conditioned.SetConditions(c...)
	_ = fieldpath.Pave(cr.Object).SetValue("status.conditions", conditioned.Conditions)
}

// GetProviderConfigReference of this Managed resource.
func (cr *Unstructured) GetProviderConfigReference() *xpv1.Reference {
	out := &xpv1.Reference{}
	if err := fieldpath.Pave(cr.Object).GetValueInto("spec.providerConfigRef", out); err != nil {
		return nil
	}
	return out
}

// SetProviderConfigReference of this Managed resource.
func (cr *Unstructured) SetProviderConfigReference(p *xpv1.Reference) {
	_ = fieldpath.Pave(cr.Object).SetValue("spec.providerConfigRef", p)
}

// GetWriteConnectionSecretToReference of this Managed resource.
func (cr *Unstructured) GetWriteConnectionSecretToReference() *xpv1.SecretReference {
	out := &xpv1.SecretReference{}
	if err := fieldpath.Pave(cr.Object).GetValueInto("spec.writeConnectionSecretToRef", out); err != nil {
		return nil
	}
	return out
}

// SetWriteConnectionSecretToReference of this Managed resource.
func (cr *Unstructured) SetWriteConnectionSecretToReference(r *xpv1.SecretReference) {
	_ = fieldpath.Pave(cr.Object).SetValue("spec.writeConnectionSecretToRef", r)
}

// GetPublishConnectionDetailsTo of this Managed resource.
func (cr *Unstructured) GetPublishConnectionDetailsTo() *xpv1.PublishConnectionDetailsTo {
	out := &xpv1.PublishConnectionDetailsTo{}
	if err := fieldpath.Pave(cr.Object).GetValueInto("spec.publishConnectionDetailsTo", out); err != nil {
		return nil
	}
	return out
}

// SetPublishConnectionDetailsTo of this Managed resource.
func (cr *Unstructured) SetPublishConnectionDetailsTo(ref *xpv1.PublishConnectionDetailsTo) {
	_ = fieldpath.Pave(cr.Object).SetValue("spec.publishConnectionDetailsTo", ref)
}

// GetManagementPolicies of this Managed resource.
func (cr *Unstructured) GetManagementPolicies() xpv1.ManagementPolicies {
	var policies xpv1.ManagementPolicies
	if err := fieldpath.Pave(cr.Object).GetValueInto("spec.managementPolicies", &policies); err != nil {
		return xpv1.ManagementPolicies{}
	}
	return policies
}

// SetManagementPolicies of this Managed resource.
func (cr *Unstructured) SetManagementPolicies(p xpv1.ManagementPolicies) {
	_ = fieldpath.Pave(cr.Object).SetValue("spec.managementPolicies", p)
}

// GetDeletionPolicy of this Managed resource.
func (cr *Unstructured) GetDeletionPolicy() xpv1.DeletionPolicy {
	var dp xpv1.DeletionPolicy
	if err := fieldpath.Pave(cr.Object).GetValueInto("spec.deletionPolicy", &dp); err != nil {
		// Default to Delete if not specified.
		return xpv1.DeletionDelete
	}
	return dp
}

// SetDeletionPolicy of this Managed resource.
func (cr *Unstructured) SetDeletionPolicy(p xpv1.DeletionPolicy) {
	_ = fieldpath.Pave(cr.Object).SetValue("spec.deletionPolicy", p)
}

// SetObservedGeneration of this Managed resource.
func (cr *Unstructured) SetObservedGeneration(generation int64) {
	status := &xpv1.ObservedStatus{}
	_ = fieldpath.Pave(cr.Object).GetValueInto("status", status)
	status.SetObservedGeneration(generation)
	_ = fieldpath.Pave(cr.Object).SetValue("status.observedGeneration", status.ObservedGeneration)
}

// GetObservedGeneration of this Managed resource.
func (cr *Unstructured) GetObservedGeneration() int64 {
	status := &xpv1.ObservedStatus{}
	_ = fieldpath.Pave(cr.Object).GetValueInto("status", status)
	return status.GetObservedGeneration()
}

// An ListOption modifies an unstructured list of managed resources.
type ListOption func(*UnstructuredList)

// FromReferenceToList returns a ListOption that propagates the metadata in the
// supplied reference to an unstructured list managed resource.
func FromReferenceToList(ref corev1.ObjectReference) ListOption {
	return func(list *UnstructuredList) {
		list.SetAPIVersion(ref.APIVersion)
		list.SetKind(ref.Kind + "List")
	}
}

// NewList returns a new unstructured list of managed resources.
func NewList(opts ...ListOption) *UnstructuredList {
	cr := &UnstructuredList{unstructured.UnstructuredList{Object: make(map[string]any)}}
	for _, f := range opts {
		f(cr)
	}
	return cr
}

// An UnstructuredList of managed resources.
type UnstructuredList struct {
	unstructured.UnstructuredList
}

// GetUnstructuredList returns the underlying *unstructured.UnstructuredList.
func (cr *UnstructuredList) GetUnstructuredList() *unstructured.UnstructuredList {
	return &cr.UnstructuredList
}
