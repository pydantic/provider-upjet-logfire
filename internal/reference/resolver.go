// Package reference resolves managed resource references.
package reference

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewResolver returns a reference resolver that saves the fields it resolves
// with a JSON merge patch guarded by the resource version.
//
// crossplane-runtime's default resolver saves them with a forced server-side
// apply that holds only the fields it changed. Server-side apply drops the
// fields the same manager set earlier, so an Alert edit that re-resolves its
// channels removes the projectId resolved before, and the reverse. The Alert
// then fails validation on every reconcile and cannot be deleted. A merge patch
// changes only the fields it names and drops nothing.
func NewResolver(c client.Client) managed.ReferenceResolver {
	return managed.ReferenceResolverFn(func(ctx context.Context, mg resource.Managed) error {
		rr, ok := mg.(interface {
			ResolveReferences(ctx context.Context, r client.Reader) error
		})
		if !ok {
			return nil
		}
		existing, ok := mg.DeepCopyObject().(client.Object)
		if !ok {
			return errors.New("cannot copy the managed resource")
		}
		if err := rr.ResolveReferences(ctx, c); err != nil {
			return errors.Wrap(err, "cannot resolve references")
		}
		if cmp.Equal(existing, mg, cmpopts.EquateEmpty()) {
			return nil
		}
		patch := client.MergeFromWithOptions(existing, client.MergeFromWithOptimisticLock{})
		return errors.Wrap(c.Patch(ctx, mg, patch), "cannot patch the managed resource with resolved references")
	})
}
