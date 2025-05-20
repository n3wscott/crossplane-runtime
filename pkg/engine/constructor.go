package engine

import (
	"context"
	"io"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/resource/unstructured"
	corev1 "k8s.io/api/core/v1"
	kcache "k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

func NewEngineFromManager(ctx context.Context, mgr ctrl.Manager, log logging.Logger) (*ControllerEngine, error) {
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

	go func() {
		// Don't start the cache until the manager is elected.
		<-mgr.Elected()

		if err := ca.Start(ctx); err != nil {
			log.Info("API extensions cache returned an error", "error", err)
		}

		log.Info("API extensions cache stopped")
	}()

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

	cem := NewPrometheusMetrics()
	metrics.Registry.MustRegister(cem)

	// It's important the engine's client is wrapped with unstructured.NewClient
	// because controller-runtime always caches *unstructured.Unstructured, not
	// our wrapper types like *composite.Unstructured. This client takes care of
	// automatically wrapping and unwrapping *unstructured.Unstructured.
	ce := New(mgr,
		TrackInformers(ca, mgr.GetScheme()),
		unstructured.NewClient(cached),
		unstructured.NewClient(uncached),
		WithLogger(log),
		WithMetrics(cem),
	)

	return ce, nil
}
