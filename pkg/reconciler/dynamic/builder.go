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

// Package dynamic contains components for dynamic resource reconciliation.
package dynamic

import (
	"context"
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	managedpkg "github.com/crossplane/crossplane-runtime/pkg/resource/unstructured/managed"

	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// ManagedBuilder builds controllers for managed.Unstructured resources.
// It provides a fluent API similar to controller-runtime's Builder.
type ManagedBuilder struct {
	name                   string
	gvks                   []schema.GroupVersionKind
	mgr                    ctrl.Manager
	maxConcurrentReconciles int
	predicates             []predicate.Predicate
	log                    logging.Logger
}

// ManagedOption is used to configure a ManagedBuilder.
type ManagedOption func(*ManagedBuilder)

// WithLogger sets the logger for the ManagedBuilder.
func WithLogger(log logging.Logger) ManagedOption {
	return func(b *ManagedBuilder) {
		b.log = log
	}
}

// WithPredicates sets the predicates for event filtering.
func WithPredicates(p ...predicate.Predicate) ManagedOption {
	return func(b *ManagedBuilder) {
		b.predicates = append(b.predicates, p...)
	}
}

// NewManagedBuilder creates a new ManagedBuilder.
func NewManagedBuilder(mgr ctrl.Manager, opts ...ManagedOption) *ManagedBuilder {
	b := &ManagedBuilder{
		mgr:                    mgr,
		maxConcurrentReconciles: 1,
		log:                    logging.NewNopLogger(),
	}

	for _, o := range opts {
		o(b)
	}

	return b
}

// Named sets the name of the controller.
func (b *ManagedBuilder) Named(name string) *ManagedBuilder {
	b.name = name
	return b
}

// ForKind adds a GVK to be watched by the controller.
func (b *ManagedBuilder) ForKind(gvk schema.GroupVersionKind) *ManagedBuilder {
	b.gvks = append(b.gvks, gvk)
	return b
}

// WithOptions sets controller options.
func (b *ManagedBuilder) WithOptions(options controller.Options) *ManagedBuilder {
	b.maxConcurrentReconciles = options.MaxConcurrentReconciles
	return b
}

// Complete builds and registers the controller with the manager.
func (b *ManagedBuilder) Complete(r reconcile.Reconciler) error {
	if b.name == "" {
		return errors.New("must specify Controller name")
	}

	if len(b.gvks) == 0 {
		return errors.New("must specify at least one GroupVersionKind to watch")
	}

	ctrl, err := controller.New(b.name, b.mgr, controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: b.maxConcurrentReconciles,
	})
	if err != nil {
		return errors.Wrap(err, "failed to create controller")
	}

	// Set up watches for all GVKs
	for _, gvk := range b.gvks {
		obj := managedpkg.New(managedpkg.WithGroupVersionKind(gvk))
		
		// Set up watch for this GVK
		if err := ctrl.Watch(
			&source.Kind{Type: obj},
			&handler.EnqueueRequestForObject{},
			b.predicates...,
		); err != nil {
			return errors.Wrapf(err, "failed to watch GVK %s", gvk.String())
		}
		
		b.log.Debug("Added watch for GVK", "controller", b.name, "gvk", gvk.String())
	}

	return nil
}

// BuildAndSetupWithManager builds and registers the controller with the manager.
func (b *ManagedBuilder) BuildAndSetupWithManager(r managed.Reconciler) error {
	return b.Complete(r)
}

// GVKWatcher watches resources by GVK.
type GVKWatcher struct {
	gvk     schema.GroupVersionKind
	handler handler.EventHandler
}

// Start starts the watcher.
func (w *GVKWatcher) Start(ctx context.Context, handler reconcile.Reconciler) error {
	// To be implemented
	return nil
}

// ManagedReconcilerBuilder builds a managed.Reconciler and its controller.
type ManagedReconcilerBuilder struct {
	builder      *ManagedBuilder
	mgr          ctrl.Manager
	forKind      schema.GroupVersionKind
	pollInterval resource.ManagedConnectionPollDuration
	logger       logging.Logger
	connecter    managed.ExternalConnecter
	recorder     event.Recorder
}

// NewManagedReconcilerBuilder creates a new ManagedReconcilerBuilder.
func NewManagedReconcilerBuilder(mgr ctrl.Manager, gvk schema.GroupVersionKind) *ManagedReconcilerBuilder {
	return &ManagedReconcilerBuilder{
		builder:   NewManagedBuilder(mgr),
		mgr:       mgr,
		forKind:   gvk,
		logger:    logging.NewNopLogger(),
	}
}

// WithLogger sets the logger for the ManagedReconcilerBuilder.
func (b *ManagedReconcilerBuilder) WithLogger(log logging.Logger) *ManagedReconcilerBuilder {
	b.logger = log
	return b
}

// WithRecorder sets the event recorder for the ManagedReconcilerBuilder.
func (b *ManagedReconcilerBuilder) WithRecorder(er event.Recorder) *ManagedReconcilerBuilder {
	b.recorder = er
	return b
}

// WithConnector sets the external connector for the ManagedReconcilerBuilder.
func (b *ManagedReconcilerBuilder) WithConnector(c managed.ExternalConnecter) *ManagedReconcilerBuilder {
	b.connecter = c
	return b
}

// WithPollInterval sets the polling interval for the ManagedReconcilerBuilder.
func (b *ManagedReconcilerBuilder) WithPollInterval(d resource.ManagedConnectionPollDuration) *ManagedReconcilerBuilder {
	b.pollInterval = d
	return b
}

// Named sets the name of the controller.
func (b *ManagedReconcilerBuilder) Named(name string) *ManagedReconcilerBuilder {
	b.builder.Named(name)
	return b
}

// WithOptions sets controller options.
func (b *ManagedReconcilerBuilder) WithOptions(options controller.Options) *ManagedReconcilerBuilder {
	b.builder.WithOptions(options)
	return b
}

// Build creates and sets up the controller with the manager.
func (b *ManagedReconcilerBuilder) Build() error {
	if b.forKind.Empty() {
		return errors.New("must specify kind to reconcile")
	}

	if b.connecter == nil {
		return errors.New("must specify external connector")
	}

	r := managed.NewReconciler(b.mgr,
		resource.ManagedKind(b.forKind),
		managed.WithLogger(b.logger),
		managed.WithRecorder(b.recorder),
		managed.WithPollInterval(b.pollInterval),
		managed.WithExternalConnecter(b.connecter),
		managed.WithNewManaged(func() resource.Managed {
			return managedpkg.New(managedpkg.WithGroupVersionKind(b.forKind))
		}),
	)

	return b.builder.
		ForKind(b.forKind).
		BuildAndSetupWithManager(r)
}