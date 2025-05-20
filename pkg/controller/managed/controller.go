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
	"context"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/crossplane/crossplane-runtime/pkg/engine"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
)

// DynamicControllerManager is an interface for managing managed controllers.
type DynamicControllerManager interface {
	// Start begins running the controllers.
	Start(ctx context.Context) error
}

// ControllerOption configures a DynamicControllerBuilder.
type ControllerOption func(*DynamicControllerBuilder)

// WithMetricsAddress sets the address for the metrics server.
func WithMetricsAddress(addr string) ControllerOption {
	return func(b *DynamicControllerBuilder) {
		b.metricsAddr = addr
	}
}

// WithHealthProbeAddress sets the address for the health probe server.
func WithHealthProbeAddress(addr string) ControllerOption {
	return func(b *DynamicControllerBuilder) {
		b.probeAddr = addr
	}
}

// WithLeaderElection enables or disables leader election.
func WithLeaderElection(enabled bool) ControllerOption {
	return func(b *DynamicControllerBuilder) {
		b.leaderElection = enabled
	}
}

// WithPollInterval sets the polling interval for all reconcilers.
func WithPollInterval(d time.Duration) ControllerOption {
	return func(b *DynamicControllerBuilder) {
		b.pollInterval = d
	}
}

// WithMaxReconcileRate sets the maximum reconciliation rate.
func WithMaxReconcileRate(rate int) ControllerOption {
	return func(b *DynamicControllerBuilder) {
		b.maxReconcileRate = rate
	}
}

// WithLogger sets the logger for the controller.
func WithLogger(log logging.Logger) ControllerOption {
	return func(b *DynamicControllerBuilder) {
		b.log = log
	}
}

// DynamicControllerBuilder builds a DynamicControllerManager.
type DynamicControllerBuilder struct {
	config           DynamicControllerConfig
	log              logging.Logger
	metricsAddr      string
	probeAddr        string
	leaderElection   bool
	pollInterval     time.Duration
	maxReconcileRate int
	engine           engine.IControllerEngine
}

// NewDynamicControllerBuilder creates a new DynamicControllerBuilder.
func NewDynamicControllerBuilder(config DynamicControllerConfig, opts ...ControllerOption) *DynamicControllerBuilder {
	b := &DynamicControllerBuilder{
		config:           config,
		log:              logging.NewNopLogger(),
		metricsAddr:      ":8080",
		probeAddr:        ":8081",
		leaderElection:   true,
		pollInterval:     1 * time.Minute,
		maxReconcileRate: 10,
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// Build creates a DynamicControllerManager from the builder.
func (b *DynamicControllerBuilder) Build(ctx context.Context) (DynamicControllerManager, error) {
	// Initialize a new scheme
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, errors.Wrap(err, "cannot add kubernetes client-go scheme")
	}

	// TODO: here we need to link the scheme of out known GKVs to the types if it is not done elsewhere.

	// Create a manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: b.metricsAddr,
		},
		HealthProbeBindAddress: b.probeAddr,
		LeaderElection:         b.leaderElection,
		LeaderElectionID:       "managed-reconciler-leader-election",
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to create controller manager")
	}

	// Setup health checks
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return nil, errors.Wrap(err, "unable to set up health check")
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return nil, errors.Wrap(err, "unable to set up ready check")
	}

	if b.engine, err = engine.NewEngineFromManager(ctx, mgr, b.log); err != nil {
		return nil, errors.Wrap(err, "unable to create engine")
	}

	// Create providers for each provider config
	var providers []*Provider
	for _, pc := range b.config.Providers {
		provider, err := NewProvider(pc,
			WithProviderLogger(b.log.WithValues("provider", pc.Name)),
			WithProviderPollInterval(b.pollInterval),
			WithProviderMaxReconcileRate(b.maxReconcileRate),
		)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot create provider %s", pc.Name)
		}
		providers = append(providers, provider)
	}

	dc := &DynamicController{
		manager:   mgr,
		engine:    b.engine,
		providers: providers,
		log:       b.log,
	}

	return dc, dc.Setup(ctx)
}

// DynamicController implements DynamicControllerManager.
type DynamicController struct {
	manager   ctrl.Manager
	engine    engine.IControllerEngine
	providers []*Provider
	log       logging.Logger
}

// Setup prepares all controllers and their manager.
func (c *DynamicController) Setup(ctx context.Context) error {
	// Set up providers with the manager
	for _, provider := range c.providers {
		if err := provider.Setup(ctx, c.engine, c.manager); err != nil {
			return errors.Wrapf(err, "cannot set up provider %s", provider.config.Name)
		}
	}
	return nil
}

// Start begins running the controllers.
func (c *DynamicController) Start(ctx context.Context) error {
	setupLog := log.Log.WithName("setup")
	setupLog.Info("starting manager")

	return c.manager.Start(ctx)
}
