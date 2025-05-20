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
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	kcache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlEvent "sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crossplane/crossplane-runtime/pkg/test"
)

type MockInformer struct {
	cache.Informer
	MockAddEventHandler    func(handler kcache.ResourceEventHandler) (kcache.ResourceEventHandlerRegistration, error)
	MockRemoveEventHandler func(id kcache.ResourceEventHandlerRegistration) error
	MockIsStopped          func() bool
}

func (m *MockInformer) AddEventHandler(handler kcache.ResourceEventHandler) (kcache.ResourceEventHandlerRegistration, error) {
	if m.MockAddEventHandler != nil {
		return m.MockAddEventHandler(handler)
	}
	return nil, nil
}

func (m *MockInformer) RemoveEventHandler(id kcache.ResourceEventHandlerRegistration) error {
	if m.MockRemoveEventHandler != nil {
		return m.MockRemoveEventHandler(id)
	}
	return nil
}

func (m *MockInformer) IsStopped() bool {
	if m.MockIsStopped != nil {
		return m.MockIsStopped()
	}
	return false
}

type MockHandler struct {
	handler.EventHandler
	CreateCalled bool
	UpdateCalled bool
	DeleteCalled bool
}

func (m *MockHandler) Create(ctx context.Context, evt ctrlEvent.CreateEvent, q workqueue.RateLimitingInterface) {
	m.CreateCalled = true
}

func (m *MockHandler) Update(ctx context.Context, evt ctrlEvent.UpdateEvent, q workqueue.RateLimitingInterface) {
	m.UpdateCalled = true
}

func (m *MockHandler) Delete(ctx context.Context, evt ctrlEvent.DeleteEvent, q workqueue.RateLimitingInterface) {
	m.DeleteCalled = true
}

func (m *MockHandler) Generic(ctx context.Context, evt ctrlEvent.GenericEvent, q workqueue.RateLimitingInterface) {
}

type MockPredicate struct {
	predicate.Predicate
	CreateResult bool
	UpdateResult bool
	DeleteResult bool
}

func (m *MockPredicate) Create(e ctrlEvent.CreateEvent) bool {
	return m.CreateResult
}

func (m *MockPredicate) Update(e ctrlEvent.UpdateEvent) bool {
	return m.UpdateResult
}

func (m *MockPredicate) Delete(e ctrlEvent.DeleteEvent) bool {
	return m.DeleteResult
}

func (m *MockPredicate) Generic(e ctrlEvent.GenericEvent) bool {
	return true
}

func TestStoppableSourceStart(t *testing.T) {
	errBoom := errors.New("boom")
	reg := &kcache.ResourceEventHandlerRegistration{}

	type fields struct {
		inf        cache.Informer
		handler    handler.EventHandler
		predicates []predicate.Predicate
	}
	type args struct {
		ctx context.Context
		q   workqueue.RateLimitingInterface
	}
	cases := map[string]struct {
		fields fields
		args   args
		want   error
	}{
		"SuccessfulStart": {
			fields: fields{
				inf: &MockInformer{
					MockAddEventHandler: func(handler kcache.ResourceEventHandler) (kcache.ResourceEventHandlerRegistration, error) {
						return reg, nil
					},
				},
				handler: &MockHandler{},
			},
			args: args{
				ctx: context.Background(),
				q:   workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
			},
			want: nil,
		},
		"FailedStart": {
			fields: fields{
				inf: &MockInformer{
					MockAddEventHandler: func(handler kcache.ResourceEventHandler) (kcache.ResourceEventHandlerRegistration, error) {
						return nil, errBoom
					},
				},
				handler: &MockHandler{},
			},
			args: args{
				ctx: context.Background(),
				q:   workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
			},
			want: errors.New("cannot add event handler: boom"),
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s := NewStoppableSource(tc.fields.inf, tc.fields.handler, tc.fields.predicates...)
			err := s.Start(tc.args.ctx, tc.args.q)
			if diff := cmp.Diff(tc.want, err, test.EquateErrors()); diff != "" {
				t.Errorf("StoppableSource.Start() error = %v, want %v\nDiff: %s", err, tc.want, diff)
			}
		})
	}
}

func TestStoppableSourceStop(t *testing.T) {
	errBoom := errors.New("boom")
	reg := &kcache.ResourceEventHandlerRegistration{}

	type fields struct {
		inf        cache.Informer
		handler    handler.EventHandler
		predicates []predicate.Predicate
		reg        kcache.ResourceEventHandlerRegistration
	}
	type args struct {
		ctx context.Context
	}
	cases := map[string]struct {
		fields fields
		args   args
		want   error
	}{
		"SuccessfulStop": {
			fields: fields{
				inf: &MockInformer{
					MockRemoveEventHandler: func(id kcache.ResourceEventHandlerRegistration) error {
						return nil
					},
					MockIsStopped: func() bool {
						return false
					},
				},
				handler: &MockHandler{},
				reg:     reg,
			},
			args: args{
				ctx: context.Background(),
			},
			want: nil,
		},
		"AlreadyStopped": {
			fields: fields{
				inf: &MockInformer{
					MockIsStopped: func() bool {
						return true
					},
				},
				handler: &MockHandler{},
				reg:     reg,
			},
			args: args{
				ctx: context.Background(),
			},
			want: nil,
		},
		"NoRegistration": {
			fields: fields{
				inf:     &MockInformer{},
				handler: &MockHandler{},
			},
			args: args{
				ctx: context.Background(),
			},
			want: nil,
		},
		"FailedStop": {
			fields: fields{
				inf: &MockInformer{
					MockRemoveEventHandler: func(id kcache.ResourceEventHandlerRegistration) error {
						return errBoom
					},
					MockIsStopped: func() bool {
						return false
					},
				},
				handler: &MockHandler{},
				reg:     reg,
			},
			args: args{
				ctx: context.Background(),
			},
			want: errors.New("cannot remove event handler: boom"),
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s := &StoppableSource{
				inf:        tc.fields.inf,
				handler:    tc.fields.handler,
				predicates: tc.fields.predicates,
				reg:        tc.fields.reg,
			}
			err := s.Stop(tc.args.ctx)
			if diff := cmp.Diff(tc.want, err, test.EquateErrors()); diff != "" {
				t.Errorf("StoppableSource.Stop() error = %v, want %v\nDiff: %s", err, tc.want, diff)
			}
		})
	}
}

func TestResourceEventHandler(t *testing.T) {
	mockObj := &client.Object{}
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	
	t.Run("OnAddPassingPredicate", func(t *testing.T) {
		mockHandler := &MockHandler{}
		predicate := &MockPredicate{CreateResult: true}
		
		handler := &resourceEventHandler{
			ctx:        context.Background(),
			handler:    mockHandler,
			queue:      queue,
			predicates: []predicate.Predicate{predicate},
		}
		
		handler.onAdd(mockObj)
		
		if !mockHandler.CreateCalled {
			t.Error("handler.Create was not called")
		}
	})
	
	t.Run("OnAddFailingPredicate", func(t *testing.T) {
		mockHandler := &MockHandler{}
		predicate := &MockPredicate{CreateResult: false}
		
		handler := &resourceEventHandler{
			ctx:        context.Background(),
			handler:    mockHandler,
			queue:      queue,
			predicates: []predicate.Predicate{predicate},
		}
		
		handler.onAdd(mockObj)
		
		if mockHandler.CreateCalled {
			t.Error("handler.Create was called despite failing predicate")
		}
	})
	
	t.Run("OnUpdatePassingPredicate", func(t *testing.T) {
		mockHandler := &MockHandler{}
		predicate := &MockPredicate{UpdateResult: true}
		
		handler := &resourceEventHandler{
			ctx:        context.Background(),
			handler:    mockHandler,
			queue:      queue,
			predicates: []predicate.Predicate{predicate},
		}
		
		handler.onUpdate(mockObj, mockObj)
		
		if !mockHandler.UpdateCalled {
			t.Error("handler.Update was not called")
		}
	})
	
	t.Run("OnDeletePassingPredicate", func(t *testing.T) {
		mockHandler := &MockHandler{}
		predicate := &MockPredicate{DeleteResult: true}
		
		handler := &resourceEventHandler{
			ctx:        context.Background(),
			handler:    mockHandler,
			queue:      queue,
			predicates: []predicate.Predicate{predicate},
		}
		
		handler.onDelete(mockObj)
		
		if !mockHandler.DeleteCalled {
			t.Error("handler.Delete was not called")
		}
	})
}