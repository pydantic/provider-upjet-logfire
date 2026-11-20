package config

import (
	"context"
	"reflect"
	"sort"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// replacedOnChange lists, per resource, the optional fields whose change makes
// Terraform replace the resource. Upjet refuses replacements, so these keep
// late-initialization: a field removed from the spec keeps its current value
// instead of planning a replacement that fails.
var replacedOnChange = map[string]map[string]bool{
	"logfire_read_token":  {"expires_at": true},
	"logfire_write_token": {"expires_at": true},
}

// LateInitializeComputedOnly stops late-initialization from copying optional
// fields that have no server-side default into the spec. Otherwise a field
// removed from the spec is copied back from the observed resource and never
// cleared, which differs from Terraform, where removing it clears it. Computed
// fields still late-initialize, because Terraform keeps their value as well.
func LateInitializeComputedOnly() ujconfig.ResourceOption {
	return func(r *ujconfig.Resource) {
		keep := replacedOnChange[r.Name]
		var ignored []string
		var walk func(prefix string, fields map[string]*schema.Schema)
		walk = func(prefix string, fields map[string]*schema.Schema) {
			for name, s := range fields {
				path := prefix + name
				if s.Optional && !s.Computed && !keep[path] {
					ignored = append(ignored, path)
					continue
				}
				// Late-initialization fills a collection only when the spec omits
				// it, and then copies it whole. Skipping a field inside a set or a
				// list would copy elements without it, and Upjet would then ignore
				// changes at an element index, which Terraform rejects for sets. So
				// only a single nested block, such as a channel's config, is walked.
				if nested, ok := s.Elem.(*schema.Resource); ok && s.Type == schema.TypeList && s.MaxItems == 1 {
					walk(path+".", nested.Schema)
				}
			}
		}
		walk("", r.TerraformResource.Schema)
		sort.Strings(ignored)
		r.LateInitializer.IgnoredFields = append(r.LateInitializer.IgnoredFields, ignored...)
	}
}

// ResolveReferencesWithUpdate adds an initializer that resolves references
// before crossplane-runtime's reference resolver runs, so that resolver finds
// nothing to change. That resolver saves only the fields it changed, as a
// server-side apply, which drops the referenced fields it set earlier. On an
// Alert with a channel reference, a later edit then removes either projectId or
// channelAssignments, and the Alert can neither reconcile nor be deleted. This
// initializer saves the whole resolved object with an update, which drops
// nothing. A reference that does not resolve is left to the reference resolver,
// which reports it.
func ResolveReferencesWithUpdate(pc *ujconfig.Provider) {
	for _, r := range pc.Resources {
		if len(r.References) > 0 {
			r.InitializerFns = append(r.InitializerFns, newReferenceInitializer)
		}
	}
}

type referencer interface {
	resource.Managed
	ResolveReferences(ctx context.Context, c client.Reader) error
}

func newReferenceInitializer(c client.Client) managed.Initializer {
	return managed.InitializerFn(func(ctx context.Context, mg resource.Managed) error {
		if meta.WasDeleted(mg) {
			return nil
		}
		resolved, ok := mg.DeepCopyObject().(referencer)
		if !ok {
			return nil
		}
		if err := resolved.ResolveReferences(ctx, c); err != nil {
			return nil //nolint:nilerr // The reference resolver runs next and reports this error.
		}
		if cmp.Equal(mg, resolved, cmpopts.EquateEmpty()) {
			return nil
		}
		if err := c.Update(ctx, resolved); err != nil {
			return err
		}
		// Carry on with the object the API server returned. Reading it back
		// through the manager's cache could return the old spec, which the
		// reference resolver would then resolve and save as a partial apply.
		reflect.ValueOf(mg).Elem().Set(reflect.ValueOf(resolved).Elem())
		return nil
	})
}
