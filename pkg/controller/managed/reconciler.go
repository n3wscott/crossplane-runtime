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

// Package managed implements Crossplane managed resources.
package managed

import (
	"context"
	"fmt"
	"github.com/crossplane/crossplane-runtime/pkg/controller/managed/watch"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/resource/unstructured"
	"io"
	corev1 "k8s.io/api/core/v1"
	kunstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kcache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	kcontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/crossplane/crossplane-runtime/pkg/engine"
)

const (
	timeout   = 2 * time.Minute
	finalizer = "managed.apiextensions.crossplane.io"
)

// Error strings.
const (
	errGet                    = "cannot get managed resource"
	errUpdate                 = "cannot update managed resource"
	errUpdateStatus           = "cannot update managed resource status"
	errAddFinalizer           = "cannot add managed resource finalizer"
	errRemoveFinalizer        = "cannot remove managed resource finalizer"
	errSelectComp             = "cannot select Composition"
	errSelectCompUpdatePolicy = "cannot select CompositionUpdatePolicy"
	errFetchComp              = "cannot fetch Composition"
	errConfigure              = "cannot configure managed resource"
	errPublish                = "cannot publish connection details"
	errWatch                  = "cannot watch resource for changes"
	errUnpublish              = "cannot unpublish connection details"
	errValidate               = "refusing to use invalid Composition"
	errAssociate              = "cannot associate composed resources with Composition resource templates"
	errCompose                = "cannot compose resources"
	errInvalidResources       = "some resources were invalid, check events"
	errRenderCD               = "cannot render composed resource"
	errSyncResources          = "cannot sync composed resources"
	errGetClaim               = "cannot get referenced claim"
	errParseClaimRef          = "cannot parse claim reference"

	reconcilePausedMsg = "Reconciliation (including deletion) is paused via the pause annotation"
)

// A CompositionTarget is the target of a composition event or condition.
type CompositionTarget string

// Composition event and condition targets.
const (
	CompositionTargetComposite         CompositionTarget = "Composite"
	CompositionTargetCompositeAndClaim CompositionTarget = "CompositeAndClaim"
)

// ReconcilerOption is used to configure the Reconciler.
type ReconcilerOption func(*Reconciler)

// WithLogger specifies how the Reconciler should log messages.
func WithLogger(log logging.Logger) ReconcilerOption {
	return func(r *Reconciler) {
		r.log = log
	}
}

// WithRecorder specifies how the Reconciler should record Kubernetes events.
func WithRecorder(er event.Recorder) ReconcilerOption {
	return func(r *Reconciler) {
		r.record = er
	}
}

// WithPollInterval specifies how long the Reconciler should wait before
// queueing a new reconciliation after a successful reconcile. The Reconciler
// uses the interval jittered +/- 10%.
func WithPollInterval(interval time.Duration) ReconcilerOption {
	return func(r *Reconciler) {
		r.pollInterval = interval
	}
}

// WithWatchStarter specifies how the Reconciler should start watches for any
// resources it composes.
func WithWatchStarter(controllerName string, h handler.EventHandler, w WatchStarter) ReconcilerOption {
	return func(r *Reconciler) {
		r.controllerName = controllerName
		r.watchHandler = h
		r.engine = w
	}
}

// A WatchStarter can start a new watch. XR controllers use this to dynamically
// start watches when they compose new kinds of resources.
type WatchStarter interface {
	// StartWatches starts the supplied watches, if they're not running already.
	StartWatches(ctx context.Context, name string, ws ...engine.Watch) error
}

// A NopWatchStarter does nothing.
type NopWatchStarter struct{}

// StartWatches does nothing.
func (n *NopWatchStarter) StartWatches(_ context.Context, _ string, _ ...engine.Watch) error {
	return nil
}

// A WatchStarterFn is a function that can start a new watch.
type WatchStarterFn func(ctx context.Context, name string, ws ...engine.Watch) error

// StartWatches starts the supplied watches, if they're not running already.
func (fn WatchStarterFn) StartWatches(ctx context.Context, name string, ws ...engine.Watch) error {
	return fn(ctx, name, ws...)
}

// NewReconciler returns a new Reconciler of managed resources.
func NewReconciler(c, uc client.Client, of resource.ManagedKind, opts ...ReconcilerOption) *Reconciler {
	r := &Reconciler{
		client: c,

		gvk: schema.GroupVersionKind(of),

		// Dynamic watches are disabled by default.
		engine: &NopWatchStarter{},

		log:    logging.NewNopLogger(),
		record: event.NewNopRecorder(),
	}

	for _, f := range opts {
		f(r)
	}

	return r
}

func MakeEngine(mgr ctrl.Manager, log logging.Logger) (*engine.ControllerEngine, error) {
	// Claim and XR controllers are started and stopped dynamically by the
	// ControllerEngine below. When realtime compositions are enabled, they also
	// start and stop their watches (e.g. of composed resources) dynamically. To
	// do this, the ControllerEngine must have exclusive ownership of a cache.
	// This allows it to track what controllers are using the cache's informers.
	ca, err := cache.New(mgr.GetConfig(), cache.Options{
		HTTPClient: mgr.GetHTTPClient(),
		Scheme:     mgr.GetScheme(),
		Mapper:     mgr.GetRESTMapper(),
		SyncPeriod: func() *time.Duration { d := 1 * time.Hour; return &d }(), // TODO: &c.SyncInterval,

		// When a CRD is deleted, any informers for its GVKs will start trying
		// to restart their watches, and fail with scary errors. This should
		// only happen when realtime composition is enabled, and we should GC
		// the informer within 60 seconds. This handler tries to make the error
		// a little more informative, and less scary.
		DefaultWatchErrorHandler: func(_ *kcache.Reflector, err error) {
			if errors.Is(io.EOF, err) {
				// Watch closed normally.
				return
			}
			//log.Debug("Watch error - probably due to CRD being uninstalled", "error", err)
		},
	})

	if err != nil {
		return nil, errors.Wrap(err, "cannot create cache for API extension controllers")
	}
	cached, err := client.New(mgr.GetConfig(), client.Options{
		HTTPClient: mgr.GetHTTPClient(),
		Scheme:     mgr.GetScheme(),
		Mapper:     mgr.GetRESTMapper(),
		Cache: &client.CacheOptions{
			Reader: ca,

			// Don't cache secrets - there may be a lot of them.
			DisableFor: []client.Object{&corev1.Secret{}},

			// Cache unstructured resources (like XRs and MRs) on Get and List.
			Unstructured: true,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "cannot create client for API extension controllers")
	}

	// Create a separate no-cache client for use when the composite controller does not find an Unstructured
	// resource that it expects to find in the cache.
	uncached, err := client.New(mgr.GetConfig(), client.Options{
		HTTPClient: mgr.GetHTTPClient(),
		Scheme:     mgr.GetScheme(),
		Mapper:     mgr.GetRESTMapper(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "cannot create uncached client for API extension controllers")
	}

	cem := engine.NewPrometheusMetrics()
	metrics.Registry.MustRegister(cem)

	// It's important the engine's client is wrapped with unstructured.NewClient
	// because controller-runtime always caches *unstructured.Unstructured, not
	// our wrapper types like *composite.Unstructured. This client takes care of
	// automatically wrapping and unwrapping *unstructured.Unstructured.
	ce := engine.New(mgr,
		engine.TrackInformers(ca, mgr.GetScheme()),
		unstructured.NewClient(cached),
		unstructured.NewClient(uncached),
		engine.WithLogger(log),
		engine.WithMetrics(cem),
	)

	return ce, nil
}

func (r *Reconciler) Starter(ctx context.Context, mgr ctrl.Manager, gvk schema.GroupVersionKind, cr reconcile.Reconciler) error {
	ko := kcontroller.Options{} // r.options.ForControllerRuntime()

	name := fmt.Sprintf("%s.%s.%s", gvk.Kind, gvk.Group, gvk.Version)

	ko.RateLimiter = workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](1*time.Second, 30*time.Second)
	ko.Reconciler = ratelimiter.NewReconciler(name, errors.WithSilentRequeueOnConflict(cr), ratelimiter.NewGlobal(1)) // TODO: don't hard code the rate limit.

	co := []engine.ControllerOption{engine.WithRuntimeOptions(ko)}

	if err := r.engine.Start(name, co...); err != nil {
		return err
	}

	// This must be *unstructured.Unstructured, not *composite.Unstructured.
	// controller-runtime doesn't support watching types that satisfy the
	// runtime.Unstructured interface - only *unstructured.Unstructured.
	kind := &kunstructured.Unstructured{}
	kind.SetGroupVersionKind(gvk)

	if err := r.engine.StartWatches(ctx, name,
		engine.WatchFor(xr, engine.WatchTypeCompositeResource, &handler.EnqueueRequestForObject{}),
	); err != nil {
		return err
	}

	return nil
}

// A Reconciler reconciles managed resources.
type Reconciler struct {
	client client.Client
	gvk    schema.GroupVersionKind

	// Used to dynamically start composed resource watches.
	controllerName string
	engine         ControllerEngine
	watchHandler   handler.EventHandler

	log    logging.Logger
	record event.Recorder

	pollInterval time.Duration
}

// Reconcile a managed resource.
func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) { //nolint:gocognit // Reconcile methods are often very complex. Be wary.
	log := r.log.WithValues("request", req)
	log.Debug("Reconciling")
	return reconcile.Result{}, nil
}
