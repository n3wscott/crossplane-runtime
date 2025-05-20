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

package dynamic

import (
	"context"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/test"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// mockManager is a mock implementation of ctrl.Manager for testing.
type mockManager struct {
	ctrl.Manager
	addFunc func(ctrl.Runnable) error
}

func (m *mockManager) Add(r ctrl.Runnable) error {
	if m.addFunc != nil {
		return m.addFunc(r)
	}
	return nil
}

func (m *mockManager) GetEventRecorderFor(name string) record.EventRecorder {
	return nil
}

// mockController is a mock implementation of controller.Controller for testing.
type mockController struct {
	controller.Controller
	watchFunc func(source.Source, handler.EventHandler, ...predicate.Predicate) error
}

func (c *mockController) Watch(src source.Source, eventHandler handler.EventHandler, predicates ...predicate.Predicate) error {
	if c.watchFunc != nil {
		return c.watchFunc(src, eventHandler, predicates...)
	}
	return nil
}

// mockReconciler is a mock implementation of reconcile.Reconciler for testing.
type mockReconciler struct {
	reconcile.Reconciler
	reconcileFunc func(context.Context, reconcile.Request) (reconcile.Result, error)
}

func (r *mockReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	if r.reconcileFunc != nil {
		return r.reconcileFunc(ctx, req)
	}
	return reconcile.Result{}, nil
}

// mockNewController mocks the controller.New function for testing.
func mockNewController(name string, mgr ctrl.Manager, options controller.Options) (controller.Controller, error) {
	return &mockController{}, nil
}

func TestManagedBuilder_Complete(t *testing.T) {
	type fields struct {
		name                   string
		gvks                   []schema.GroupVersionKind
		mgr                    ctrl.Manager
		maxConcurrentReconciles int
	}
	type args struct {
		r reconcile.Reconciler
	}
	type want struct {
		err error
	}
	cases := map[string]struct {
		fields fields
		args   args
		want   want
	}{
		"MissingControllerName": {
			fields: fields{
				gvks: []schema.GroupVersionKind{
					{Group: "example.com", Version: "v1", Kind: "Widget"},
				},
				mgr: &mockManager{},
			},
			args: args{
				r: &mockReconciler{},
			},
			want: want{
				err: errors.New("must specify Controller name"),
			},
		},
		"NoGVKs": {
			fields: fields{
				name: "test-controller",
				mgr:  &mockManager{},
			},
			args: args{
				r: &mockReconciler{},
			},
			want: want{
				err: errors.New("must specify at least one GroupVersionKind to watch"),
			},
		},
		"Success": {
			fields: fields{
				name: "test-controller",
				gvks: []schema.GroupVersionKind{
					{Group: "example.com", Version: "v1", Kind: "Widget"},
				},
				mgr:                    &mockManager{},
				maxConcurrentReconciles: 2,
			},
			args: args{
				r: &mockReconciler{},
			},
			want: want{
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := &ManagedBuilder{
				name:                   tc.fields.name,
				gvks:                   tc.fields.gvks,
				mgr:                    tc.fields.mgr,
				maxConcurrentReconciles: tc.fields.maxConcurrentReconciles,
				log:                    logging.NewNopLogger(),
			}
			err := b.Complete(tc.args.r)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("ManagedBuilder.Complete() error = %v, want %v\n%s", err, tc.want.err, diff)
			}
		})
	}
}

func TestManagedReconcilerBuilder_Build(t *testing.T) {
	type fields struct {
		builder      *ManagedBuilder
		mgr          ctrl.Manager
		forKind      schema.GroupVersionKind
		pollInterval resource.ManagedConnectionPollDuration
		logger       logging.Logger
		connecter    managed.ExternalConnecter
	}
	type want struct {
		err error
	}
	cases := map[string]struct {
		fields fields
		want   want
	}{
		"MissingGVK": {
			fields: fields{
				builder:   NewManagedBuilder(&mockManager{}),
				mgr:       &mockManager{},
				connecter: &test.MockConnector{},
			},
			want: want{
				err: errors.New("must specify kind to reconcile"),
			},
		},
		"MissingConnector": {
			fields: fields{
				builder: NewManagedBuilder(&mockManager{}),
				mgr:     &mockManager{},
				forKind: schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Widget"},
			},
			want: want{
				err: errors.New("must specify external connector"),
			},
		},
		"Success": {
			fields: fields{
				builder:   NewManagedBuilder(&mockManager{}),
				mgr:       &mockManager{},
				forKind:   schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Widget"},
				connecter: &test.MockConnector{},
				logger:    logging.NewNopLogger(),
			},
			want: want{
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := &ManagedReconcilerBuilder{
				builder:      tc.fields.builder,
				mgr:          tc.fields.mgr,
				forKind:      tc.fields.forKind,
				pollInterval: tc.fields.pollInterval,
				logger:       tc.fields.logger,
				connecter:    tc.fields.connecter,
			}
			err := b.Build()
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("ManagedReconcilerBuilder.Build() error = %v, want %v\n%s", err, tc.want.err, diff)
			}
		})
	}
}

func TestNewManagedBuilder(t *testing.T) {
	mockMgr := &mockManager{}
	builder := NewManagedBuilder(mockMgr)

	if builder.mgr != mockMgr {
		t.Errorf("NewManagedBuilder() did not set manager correctly, got = %v, want %v", builder.mgr, mockMgr)
	}

	if builder.maxConcurrentReconciles != 1 {
		t.Errorf("NewManagedBuilder() did not set default maxConcurrentReconciles, got = %v, want %v", 
			builder.maxConcurrentReconciles, 1)
	}
}

func TestManagedBuilder_Named(t *testing.T) {
	builder := &ManagedBuilder{}
	name := "test-controller"
	
	result := builder.Named(name)
	
	if result != builder {
		t.Errorf("Named() did not return the builder")
	}
	
	if builder.name != name {
		t.Errorf("Named() did not set the name correctly, got = %v, want %v", builder.name, name)
	}
}

func TestManagedBuilder_ForKind(t *testing.T) {
	builder := &ManagedBuilder{}
	gvk := schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Widget"}
	
	result := builder.ForKind(gvk)
	
	if result != builder {
		t.Errorf("ForKind() did not return the builder")
	}
	
	if len(builder.gvks) != 1 || builder.gvks[0] != gvk {
		t.Errorf("ForKind() did not add the GVK correctly, got = %v, want %v", builder.gvks, []schema.GroupVersionKind{gvk})
	}
}