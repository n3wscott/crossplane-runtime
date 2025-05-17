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
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/external/client"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

// ProviderOption configures a Provider.
type ProviderOption func(*Provider)

// WithProviderLogger sets the logger for the Provider.
func WithProviderLogger(log logging.Logger) ProviderOption {
	return func(p *Provider) {
		p.log = log
	}
}

// WithProviderPollInterval sets the polling interval for reconciliations.
func WithProviderPollInterval(interval time.Duration) ProviderOption {
	return func(p *Provider) {
		p.pollInterval = interval
	}
}

// WithMaxReconcileRate sets the maximum reconciliation rate.
func WithProviderMaxReconcileRate(rate int) ProviderOption {
	return func(p *Provider) {
		p.maxReconcileRate = rate
	}
}

// Provider represents a dynamic provider connection and its configuration.
type Provider struct {
	config           ProviderConfig
	connector        *client.StreamingConnector
	log              logging.Logger
	gvks             []schema.GroupVersionKind
	pollInterval     time.Duration
	maxReconcileRate int
}

// NewProvider creates a new Provider with the given configuration and options.
func NewProvider(config ProviderConfig, opts ...ProviderOption) (*Provider, error) {
	p := &Provider{
		config:           config,
		log:              logging.NewNopLogger(),
		pollInterval:     1 * time.Minute,
		maxReconcileRate: 10,
	}

	for _, opt := range opts {
		opt(p)
	}

	// Parse GVKs for all resource types
	for _, rt := range config.ResourceTypes {
		gvk, err := ResourceTypeToGVK(rt)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid API version %s", rt.APIVersion)
		}
		p.gvks = append(p.gvks, gvk)
	}

	// Create the streaming connector
	var creds credentials.TransportCredentials
	if config.UseSSL {
		// In a real implementation, we'd load proper TLS credentials
		// This is just a placeholder
		p.log.Debug("SSL is enabled, but insecure credentials are being used for demonstration")
		creds = insecure.NewCredentials()
	} else {
		creds = insecure.NewCredentials()
	}

	p.connector = client.NewStreamingConnector(
		config.Endpoint,
		creds,
		client.WithClientLogger(p.log),
		client.WithResourceTypes(p.gvks...),
	)

	return p, nil
}

// Setup sets up controllers for all resource types handled by this provider.
func (p *Provider) Setup(mgr ctrl.Manager) error {
	// Add a cleanup function to close the connector when the manager stops
	mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		<-ctx.Done() // Wait for context to be cancelled (manager is stopping)
		return p.connector.Close()
	}))

	// Set up a controller for each resource type
	for _, rt := range p.config.ResourceTypes {
		if err := p.setupResourceController(mgr, rt); err != nil {
			return err
		}
	}

	return nil
}

// setupResourceController sets up a controller for a specific resource type.
func (p *Provider) setupResourceController(mgr ctrl.Manager, rt ResourceType) error {
	gvk, err := ResourceTypeToGVK(rt)
	if err != nil {
		return err
	}

	// Ensure the schema knows about this type
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(gvk)

	// Add the type to the scheme
	mgr.GetScheme().AddKnownTypeWithName(gvk, u.DeepCopyObject())

	// Set up the controller name
	gv, _ := schema.ParseGroupVersion(rt.APIVersion)
	name := fmt.Sprintf("%s.%s.%s", rt.Kind, gv.Group, p.config.Name)

	// Create the reconciler
	r := managed.NewReconciler(mgr,
		resource.ManagedKind(gvk),
		managed.WithLogger(p.log.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithPollInterval(p.pollInterval),
		managed.WithExternalConnecter(p.connector),
	)

	// Create the unstructured object with the correct GVK
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)

	// Setup the controller
	if err := ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(obj).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: p.maxReconcileRate,
		}).
		Complete(r); err != nil {
		return errors.Wrapf(err, "cannot set up controller for %s", gvk)
	}

	p.log.Debug("Set up controller", "gvk", gvk.String())
	return nil
}
