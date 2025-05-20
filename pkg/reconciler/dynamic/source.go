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

	"github.com/crossplane/crossplane-runtime/pkg/errors"

	kcache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlEvent "sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var _ source.TypedSource[reconcile.Request] = &StoppableSource{}

// NewStoppableSource returns a new watch source that can be stopped.
func NewStoppableSource(inf cache.Informer, h handler.EventHandler, ps ...predicate.Predicate) *StoppableSource {
	return &StoppableSource{inf: inf, handler: h, predicates: ps}
}

// A StoppableSource is a controller-runtime watch source that can be stopped.
type StoppableSource struct {
	inf cache.Informer

	handler    handler.EventHandler
	predicates []predicate.Predicate

	reg kcache.ResourceEventHandlerRegistration
}

// Start is internal and should be called only by the Controller to register
// an EventHandler with the Informer to enqueue reconcile.Requests.
func (s *StoppableSource) Start(ctx context.Context, q workqueue.TypedRateLimitingInterface[reconcile.Request]) error {
	// Create wrapper event handler that applies predicates
	eventHandler := &resourceEventHandler{
		ctx:        ctx, 
		handler:    s.handler,
		queue:      q,
		predicates: s.predicates,
	}

	// Add the event handler to the informer
	reg, err := s.inf.AddEventHandler(eventHandler.handlerFuncs())
	if err != nil {
		return errors.Wrap(err, "cannot add event handler")
	}
	s.reg = reg

	return nil
}

// Stop removes the EventHandler from the source's Informer.
func (s *StoppableSource) Stop(ctx context.Context) error {
	if s.reg == nil || s.inf.IsStopped() {
		return nil
	}

	if err := s.inf.RemoveEventHandler(s.reg); err != nil {
		return errors.Wrap(err, "cannot remove event handler")
	}

	s.reg = nil
	return nil
}

// resourceEventHandler handles events from the informer and applies predicates.
type resourceEventHandler struct {
	ctx        context.Context
	handler    handler.EventHandler
	queue      workqueue.TypedRateLimitingInterface[reconcile.Request]
	predicates []predicate.Predicate
}

// handlerFuncs returns the ResourceEventHandlerFuncs.
func (e *resourceEventHandler) handlerFuncs() kcache.ResourceEventHandlerFuncs {
	return kcache.ResourceEventHandlerFuncs{
		AddFunc:    e.onAdd,
		UpdateFunc: e.onUpdate,
		DeleteFunc: e.onDelete,
	}
}

// onAdd handles object creation events.
func (e *resourceEventHandler) onAdd(obj interface{}) {
	o, ok := obj.(client.Object)
	if !ok {
		return
	}

	// Create the event
	evt := ctrlEvent.TypedCreateEvent[client.Object]{Object: o}

	// Apply predicates
	for _, p := range e.predicates {
		if !p.Create(evt) {
			return
		}
	}

	// Handle the event
	ctx, cancel := context.WithCancel(e.ctx)
	defer cancel()
	
	if typedHandler, ok := e.handler.(handler.TypedEventHandler[client.Object, reconcile.Request]); ok {
		typedHandler.Create(ctx, evt, e.queue)
	} else {
		// Fallback for non-typed handlers
		untyped := ctrlEvent.CreateEvent{Object: o}
		e.handler.Create(ctx, untyped, e.queue)
	}
}

// onUpdate handles object update events.
func (e *resourceEventHandler) onUpdate(oldObj, newObj interface{}) {
	oldObject, ok := oldObj.(client.Object)
	if !ok {
		return
	}

	newObject, ok := newObj.(client.Object)
	if !ok {
		return
	}

	// Create the typed event
	evt := ctrlEvent.TypedUpdateEvent[client.Object]{ObjectOld: oldObject, ObjectNew: newObject}

	// Apply predicates
	for _, p := range e.predicates {
		if !p.Update(evt) {
			return
		}
	}

	// Handle the event
	ctx, cancel := context.WithCancel(e.ctx)
	defer cancel()
	
	if typedHandler, ok := e.handler.(handler.TypedEventHandler[client.Object, reconcile.Request]); ok {
		typedHandler.Update(ctx, evt, e.queue)
	} else {
		// Fallback for non-typed handlers
		untyped := ctrlEvent.UpdateEvent{ObjectOld: oldObject, ObjectNew: newObject}
		e.handler.Update(ctx, untyped, e.queue)
	}
}

// onDelete handles object deletion events.
func (e *resourceEventHandler) onDelete(obj interface{}) {
	var typedEvt ctrlEvent.TypedDeleteEvent[client.Object]
	var untypedEvt ctrlEvent.DeleteEvent

	switch o := obj.(type) {
	case client.Object:
		typedEvt = ctrlEvent.TypedDeleteEvent[client.Object]{Object: o}
		untypedEvt = ctrlEvent.DeleteEvent{Object: o}
	// Handle tombstone events
	case kcache.DeletedFinalStateUnknown:
		if object, ok := o.Obj.(client.Object); ok {
			typedEvt = ctrlEvent.TypedDeleteEvent[client.Object]{
				Object:             object,
				DeleteStateUnknown: true,
			}
			untypedEvt = ctrlEvent.DeleteEvent{
				Object:             object,
				DeleteStateUnknown: true,
			}
		} else {
			return
		}
	default:
		return
	}

	// Apply predicates
	for _, p := range e.predicates {
		if !p.Delete(typedEvt) {
			return
		}
	}

	// Handle the event
	ctx, cancel := context.WithCancel(e.ctx)
	defer cancel()
	
	if typedHandler, ok := e.handler.(handler.TypedEventHandler[client.Object, reconcile.Request]); ok {
		typedHandler.Delete(ctx, typedEvt, e.queue)
	} else {
		// Fallback for non-typed handlers
		e.handler.Delete(ctx, untypedEvt, e.queue)
	}
}