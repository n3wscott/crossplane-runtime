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
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	managedpkg "github.com/crossplane/crossplane-runtime/pkg/resource/unstructured/managed"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// DynamicSource is a source that watches a specific GVK using dynamic client.
type DynamicSource struct {
	gvk        schema.GroupVersionKind
	client     dynamic.Interface
	mapper     meta.RESTMapper
	handler    handler.EventHandler
	predicates []predicate.Predicate
	log        logging.Logger
}

// DynamicSourceOption configures a DynamicSource.
type DynamicSourceOption func(*DynamicSource)

// WithSourceLogger sets the logger for the DynamicSource.
func WithSourceLogger(log logging.Logger) DynamicSourceOption {
	return func(s *DynamicSource) {
		s.log = log
	}
}

// WithSourcePredicates sets the predicates for the DynamicSource.
func WithSourcePredicates(predicates ...predicate.Predicate) DynamicSourceOption {
	return func(s *DynamicSource) {
		s.predicates = predicates
	}
}

// NewDynamicSource creates a new DynamicSource.
func NewDynamicSource(gvk schema.GroupVersionKind, client dynamic.Interface, mapper meta.RESTMapper, handler handler.EventHandler, opts ...DynamicSourceOption) *DynamicSource {
	s := &DynamicSource{
		gvk:     gvk,
		client:  client,
		mapper:  mapper,
		handler: handler,
		log:     logging.NewNopLogger(),
	}

	for _, o := range opts {
		o(s)
	}

	return s
}

// Start starts the DynamicSource.
func (s *DynamicSource) Start(ctx context.Context, queue workqueue.RateLimitingInterface) error {
	mapping, err := s.mapper.RESTMapping(s.gvk.GroupKind(), s.gvk.Version)
	if err != nil {
		return errors.Wrapf(err, "failed to get REST mapping for GVK %s", s.gvk.String())
	}

	// Create a dynamic informer for the specified GVK
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(
		s.client,
		0, // No resync
		"", // All namespaces
		nil,
	)
	informer := factory.ForResource(mapping.Resource)

	// Set up event handlers
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			s.handleEvent(queue, event.CreateEvent{Object: convertToManaged(obj, s.gvk)})
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			s.handleEvent(queue, event.UpdateEvent{
				ObjectOld: convertToManaged(oldObj, s.gvk),
				ObjectNew: convertToManaged(newObj, s.gvk),
			})
		},
		DeleteFunc: func(obj interface{}) {
			s.handleEvent(queue, event.DeleteEvent{Object: convertToManaged(obj, s.gvk)})
		},
	})

	// Start the informer
	factory.Start(ctx.Done())
	
	// Wait for cache to sync
	if !cache.WaitForCacheSync(ctx.Done(), informer.Informer().HasSynced) {
		return errors.Errorf("failed to wait for cache sync for GVK %s", s.gvk.String())
	}

	s.log.Debug("Started dynamic source", "gvk", s.gvk.String())
	
	return nil
}

// convertToManaged converts an object to a managed.Unstructured.
func convertToManaged(obj interface{}, gvk schema.GroupVersionKind) runtime.Object {
	// Handle cache.DeletedFinalStateUnknown
	if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = tombstone.Obj
	}

	// Convert to unstructured
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil
	}

	// Create a new managed.Unstructured with the same data
	managed := managedpkg.New(managedpkg.WithGroupVersionKind(gvk))
	managed.Unstructured = *u
	
	return managed
}

// handleEvent handles an event.
func (s *DynamicSource) handleEvent(queue workqueue.RateLimitingInterface, evt interface{}) {
	// Apply predicates
	for _, p := range s.predicates {
		switch e := evt.(type) {
		case event.CreateEvent:
			if !p.Create(e) {
				return
			}
		case event.UpdateEvent:
			if !p.Update(e) {
				return
			}
		case event.DeleteEvent:
			if !p.Delete(e) {
				return
			}
		case event.GenericEvent:
			if !p.Generic(e) {
				return
			}
		}
	}

	// Handle the event using the provided handler
	switch e := evt.(type) {
	case event.CreateEvent:
		s.handler.Create(context.Background(), e, queue)
	case event.UpdateEvent:
		s.handler.Update(context.Background(), e, queue)
	case event.DeleteEvent:
		s.handler.Delete(context.Background(), e, queue)
	case event.GenericEvent:
		s.handler.Generic(context.Background(), e, queue)
	}
}

// MultiGVKSource is a source that watches multiple GVKs.
type MultiGVKSource struct {
	sources []*DynamicSource
}

// NewMultiGVKSource creates a new MultiGVKSource.
func NewMultiGVKSource(sources ...*DynamicSource) *MultiGVKSource {
	return &MultiGVKSource{
		sources: sources,
	}
}

// Start starts the MultiGVKSource.
func (s *MultiGVKSource) Start(ctx context.Context, queue workqueue.RateLimitingInterface) error {
	for _, src := range s.sources {
		if err := src.Start(ctx, queue); err != nil {
			return err
		}
	}
	return nil
}

// ManagedKindSource is a source.Source that watches managed.Unstructured resources.
type ManagedKindSource struct {
	gvk        schema.GroupVersionKind
	client     client.Client
	handler    handler.EventHandler
	predicates []predicate.Predicate
}

var _ source.Source = &ManagedKindSource{}

// NewManagedKindSource creates a new ManagedKindSource.
func NewManagedKindSource(gvk schema.GroupVersionKind, client client.Client, handler handler.EventHandler, predicates ...predicate.Predicate) *ManagedKindSource {
	return &ManagedKindSource{
		gvk:        gvk,
		client:     client,
		handler:    handler,
		predicates: predicates,
	}
}

// Start starts the ManagedKindSource.
func (s *ManagedKindSource) Start(ctx context.Context, handler handler.EventHandler, queue workqueue.RateLimitingInterface, prct ...predicate.Predicate) error {
	// Create a new managed.Unstructured with the specified GVK
	obj := managedpkg.New(managedpkg.WithGroupVersionKind(s.gvk))
	
	// Use controller-runtime's source.Kind to watch the GVK
	src := source.Kind{
		Type: obj,
	}
	
	// Combine predicates
	predicates := append(s.predicates, prct...)
	
	return src.Start(ctx, handler, queue, predicates...)
}

// EnqueueRequestForManagedObject is an EventHandler that enqueues reconcile.Requests
// for managed.Unstructured objects.
type EnqueueRequestForManagedObject struct{}

var _ handler.EventHandler = &EnqueueRequestForManagedObject{}

// Create implements EventHandler.
func (e *EnqueueRequestForManagedObject) Create(ctx context.Context, evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		return
	}
	q.Add(reconcile.Request{NamespacedName: client.ObjectKeyFromObject(evt.Object)})
}

// Update implements EventHandler.
func (e *EnqueueRequestForManagedObject) Update(ctx context.Context, evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	if evt.ObjectOld == nil {
		return
	}
	q.Add(reconcile.Request{NamespacedName: client.ObjectKeyFromObject(evt.ObjectOld)})
}

// Delete implements EventHandler.
func (e *EnqueueRequestForManagedObject) Delete(ctx context.Context, evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		return
	}
	q.Add(reconcile.Request{NamespacedName: client.ObjectKeyFromObject(evt.Object)})
}

// Generic implements EventHandler.
func (e *EnqueueRequestForManagedObject) Generic(ctx context.Context, evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		return
	}
	q.Add(reconcile.Request{NamespacedName: client.ObjectKeyFromObject(evt.Object)})
}