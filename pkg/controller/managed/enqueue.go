package managed

import (
	"context"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	managedpkg "github.com/crossplane/crossplane-runtime/pkg/resource/unstructured/managed"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kevent "sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func EnqueueForManaged(of resource.ManagedKind, c client.Reader, log logging.Logger) handler.Funcs {
	return handler.Funcs{
		CreateFunc: func(ctx context.Context, e kevent.CreateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			m, ok := e.Object.(*managedpkg.Unstructured)
			if !ok {
				return
			}

			q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      m.GetName(),
				Namespace: m.GetNamespace(),
			}})
		},
		// TODO: implement the other CRUD methods
	}
}
