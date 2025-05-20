package managed

import (
	"context"
	"github.com/crossplane/crossplane-runtime/pkg/engine"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// A ControllerEngine can start and stop Kubernetes controllers on demand.
//
//nolint:interfacebloat // We use this interface to stub the engine for testing, and we need all of its functionality.
type ControllerEngine interface {
	Start(name string, o ...engine.ControllerOption) error
	Stop(ctx context.Context, name string) error
	IsRunning(name string) bool
	GetWatches(name string) ([]engine.WatchID, error)
	StartWatches(ctx context.Context, name string, ws ...engine.Watch) error
	StopWatches(ctx context.Context, name string, ws ...engine.WatchID) (int, error)
	GetCached() client.Client
	GetUncached() client.Client
	GetFieldIndexer() client.FieldIndexer
}

// A NopEngine does nothing.
type NopEngine struct{}

// Start does nothing.
func (e *NopEngine) Start(_ string, _ ...engine.ControllerOption) error {
	return nil
}

// Stop does nothing.
func (e *NopEngine) Stop(_ context.Context, _ string) error { return nil }

// IsRunning always returns true.
func (e *NopEngine) IsRunning(_ string) bool { return true }

// GetWatches does nothing.
func (e *NopEngine) GetWatches(_ string) ([]engine.WatchID, error) { return nil, nil }

// StartWatches does nothing.
func (e *NopEngine) StartWatches(_ context.Context, _ string, _ ...engine.Watch) error { return nil }

// StopWatches does nothing.
func (e *NopEngine) StopWatches(_ context.Context, _ string, _ ...engine.WatchID) (int, error) {
	return 0, nil
}

// GetCached returns a nil client.
func (e *NopEngine) GetCached() client.Client {
	return nil
}

// GetUncached returns a nil client.
func (e *NopEngine) GetUncached() client.Client {
	return nil
}

// GetFieldIndexer returns a nil field indexer.
func (e *NopEngine) GetFieldIndexer() client.FieldIndexer {
	return nil
}
