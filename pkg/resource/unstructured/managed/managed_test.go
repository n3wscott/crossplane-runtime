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

package managed

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

var _ resource.Managed = &Unstructured{}

func TestWithGroupVersionKind(t *testing.T) {
	gvk := corev1.SchemeGroupVersion.WithKind("Pod")
	cases := map[string]struct {
		gvk  corev1.SchemeGroupVersion
		kind string
		want *Unstructured
	}{
		"New": {
			gvk:  corev1.SchemeGroupVersion,
			kind: "Pod",
			want: &Unstructured{
				Unstructured: unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "v1",
						"kind":       "Pod",
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := New(WithGroupVersionKind(tc.gvk.WithKind(tc.kind)))
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("New(WithGroupVersionKind(...): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestConditions(t *testing.T) {
	cases := map[string]struct {
		reason string
		u      *Unstructured
		set    []xpv1.Condition
		get    xpv1.ConditionType
		want   xpv1.Condition
	}{
		"NewCondition": {
			reason: "It should be possible to set a condition of an empty Unstructured.",
			u:      New(),
			set:    []xpv1.Condition{xpv1.Available(), xpv1.ReconcileSuccess()},
			get:    xpv1.TypeReady,
			want:   xpv1.Available(),
		},
		"ExistingCondition": {
			reason: "It should be possible to overwrite a condition that is already set.",
			u:      New(WithConditions(xpv1.Creating())),
			set:    []xpv1.Condition{xpv1.Available()},
			get:    xpv1.TypeReady,
			want:   xpv1.Available(),
		},
		"WeirdStatus": {
			reason: "It should not be possible to set a condition when status is not an object.",
			u: &Unstructured{unstructured.Unstructured{Object: map[string]any{
				"status": "wat",
			}}},
			set:  []xpv1.Condition{xpv1.Available()},
			get:  xpv1.TypeReady,
			want: xpv1.Condition{},
		},
		"WeirdStatusConditions": {
			reason: "Conditions should be overwritten if they are not an object.",
			u: &Unstructured{unstructured.Unstructured{Object: map[string]any{
				"status": map[string]any{
					"conditions": "wat",
				},
			}}},
			set:  []xpv1.Condition{xpv1.Available()},
			get:  xpv1.TypeReady,
			want: xpv1.Available(),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tc.u.SetConditions(tc.set...)
			got := tc.u.GetCondition(tc.get)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("\n%s\nu.GetCondition(%s): -want, +got:\n%s", tc.reason, tc.get, diff)
			}
		})
	}
}

func TestWriteConnectionSecretToReference(t *testing.T) {
	ref := &xpv1.SecretReference{Namespace: "ns", Name: "cool"}
	cases := map[string]struct {
		u    *Unstructured
		set  *xpv1.SecretReference
		want *xpv1.SecretReference
	}{
		"NewRef": {
			u:    New(),
			set:  ref,
			want: ref,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tc.u.SetWriteConnectionSecretToReference(tc.set)
			got := tc.u.GetWriteConnectionSecretToReference()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("\nu.GetWriteConnectionSecretToReference(): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestPublishConnectionDetailsTo(t *testing.T) {
	ref := &xpv1.PublishConnectionDetailsTo{Name: "cool", ConfigRef: &xpv1.Reference{Name: "config"}}
	cases := map[string]struct {
		u    *Unstructured
		set  *xpv1.PublishConnectionDetailsTo
		want *xpv1.PublishConnectionDetailsTo
	}{
		"NewRef": {
			u:    New(),
			set:  ref,
			want: ref,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tc.u.SetPublishConnectionDetailsTo(tc.set)
			got := tc.u.GetPublishConnectionDetailsTo()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("\nu.GetPublishConnectionDetailsTo(): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestProviderConfigReference(t *testing.T) {
	ref := &xpv1.Reference{Name: "cool"}
	cases := map[string]struct {
		u    *Unstructured
		set  *xpv1.Reference
		want *xpv1.Reference
	}{
		"NewRef": {
			u:    New(),
			set:  ref,
			want: ref,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tc.u.SetProviderConfigReference(tc.set)
			got := tc.u.GetProviderConfigReference()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("\nu.GetProviderConfigReference(): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestManagementPolicies(t *testing.T) {
	policies := xpv1.ManagementPolicies{xpv1.ManagementActionAll}
	cases := map[string]struct {
		u    *Unstructured
		set  xpv1.ManagementPolicies
		want xpv1.ManagementPolicies
	}{
		"NewPolicies": {
			u:    New(),
			set:  policies,
			want: policies,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tc.u.SetManagementPolicies(tc.set)
			got := tc.u.GetManagementPolicies()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("\nu.GetManagementPolicies(): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestDeletionPolicy(t *testing.T) {
	policy := xpv1.DeletionOrphan
	cases := map[string]struct {
		u    *Unstructured
		set  xpv1.DeletionPolicy
		want xpv1.DeletionPolicy
	}{
		"NewPolicy": {
			u:    New(),
			set:  policy,
			want: policy,
		},
		"Default": {
			u:    New(),
			want: xpv1.DeletionDelete,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.set != "" {
				tc.u.SetDeletionPolicy(tc.set)
			}
			got := tc.u.GetDeletionPolicy()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("\nu.GetDeletionPolicy(): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestObservedGeneration(t *testing.T) {
	cases := map[string]struct {
		u    *Unstructured
		want int64
	}{
		"Set": {
			u: New(func(u *Unstructured) {
				u.SetObservedGeneration(123)
			}),
			want: 123,
		},
		"NotFound": {
			u: New(),
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := tc.u.GetObservedGeneration()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("\nu.GetObservedGeneration(): -want, +got:\n%s", diff)
			}
		})
	}
}
