package config

import (
	"context"
	"slices"
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/pydantic/provider-upjet-logfire/apis/cluster"
	alertv1alpha1 "github.com/pydantic/provider-upjet-logfire/apis/cluster/alert/v1alpha1"
	channelv1alpha1 "github.com/pydantic/provider-upjet-logfire/apis/cluster/channel/v1alpha1"
	projectv1alpha1 "github.com/pydantic/provider-upjet-logfire/apis/cluster/project/v1alpha1"
)

func TestLateInitializationSkipsClearableFields(t *testing.T) {
	for name, provider := range map[string]*ujconfig.Provider{
		"cluster":    GetProvider(),
		"namespaced": GetProviderNamespaced(),
	} {
		t.Run(name, func(t *testing.T) {
			for resource, fields := range map[string]struct{ skipped, kept []string }{
				"logfire_project":                 {skipped: []string{"description"}, kept: []string{"visibility"}},
				"logfire_alert":                   {skipped: []string{"description", "environments", "instance_policy"}, kept: []string{"active"}},
				"logfire_channel":                 {skipped: []string{"config.recipients", "config.url"}, kept: []string{"active", "config.base_url"}},
				"logfire_gateway_spending_limits": {skipped: []string{"daily_usd", "monthly_usd", "weekly_usd"}},
				"logfire_gateway_endpoint":        {skipped: []string{"description"}},
				"logfire_read_token":              {kept: []string{"expires_at"}},
				"logfire_write_token":             {kept: []string{"expires_at"}},
			} {
				ignored := provider.Resources[resource].LateInitializer.IgnoredFields
				for _, f := range fields.skipped {
					if !slices.Contains(ignored, f) {
						t.Errorf("%s: late-initialization must skip %s, ignored %v", resource, f, ignored)
					}
				}
				for _, f := range fields.kept {
					if slices.Contains(ignored, f) {
						t.Errorf("%s: late-initialization must keep %s, ignored %v", resource, f, ignored)
					}
				}
			}
		})
	}
}

func TestReferenceInitializerWiredForReferencingKinds(t *testing.T) {
	for name, provider := range map[string]*ujconfig.Provider{
		"cluster":    GetProvider(),
		"namespaced": GetProviderNamespaced(),
	} {
		for resource, r := range provider.Resources {
			if got, want := len(r.InitializerFns) > 0, len(r.References) > 0; got != want {
				t.Errorf("%s %s: has reference initializer %v, has references %v", name, resource, got, want)
			}
		}
	}
}

func TestReferenceInitializerSavesWholeResolvedObject(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := cluster.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	project := &projectv1alpha1.Project{}
	project.SetName("proj")
	meta.SetExternalName(project, "project-uuid")
	channel := &channelv1alpha1.Channel{}
	channel.SetName("hook")
	meta.SetExternalName(channel, "channel-uuid")
	alert := &alertv1alpha1.Alert{}
	alert.SetName("alert")
	alert.Spec.ForProvider.ProjectIDRef = &xpv1.Reference{Name: "proj"}
	alert.Spec.ForProvider.ChannelAssignments = []alertv1alpha1.ChannelAssignmentsParameters{
		{ChannelIDRef: &xpv1.Reference{Name: "hook"}},
	}
	alert.Spec.ForProvider.Description = ptr.To("kept")

	updates := 0
	store := fake.NewClientBuilder().WithScheme(scheme).WithObjects(project, channel, alert).Build()
	// The manager's client reads the alert from a cache that can lag behind an update.
	c := interceptor.NewClient(store, interceptor.Funcs{
		Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			updates++
			return c.Update(ctx, obj, opts...)
		},
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if updates > 0 && key.Name == "alert" {
				t.Error("the initializer must not read the alert back through the cache, which may be stale")
			}
			return c.Get(ctx, key, obj, opts...)
		},
		Patch: func(context.Context, client.WithWatch, client.Object, client.Patch, ...client.PatchOption) error {
			t.Error("the initializer must save with an update, not a patch")
			return nil
		},
	})

	initialize := newReferenceInitializer(c)
	mg := &alertv1alpha1.Alert{}
	if err := store.Get(context.Background(), client.ObjectKey{Name: "alert"}, mg); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := initialize.Initialize(context.Background(), mg); err != nil {
			t.Fatal(err)
		}
	}
	if updates != 1 {
		t.Errorf("updates = %d, want 1: the second pass has nothing to resolve", updates)
	}

	stored := &alertv1alpha1.Alert{}
	if err := store.Get(context.Background(), client.ObjectKey{Name: "alert"}, stored); err != nil {
		t.Fatal(err)
	}
	p := stored.Spec.ForProvider
	if ptr.Deref(p.ProjectID, "") != "project-uuid" || len(p.ChannelAssignments) != 1 ||
		ptr.Deref(p.ChannelAssignments[0].ChannelID, "") != "channel-uuid" || ptr.Deref(p.Description, "") != "kept" {
		t.Errorf("stored spec = %+v, want both references resolved and other fields kept", p)
	}
	if ptr.Deref(mg.Spec.ForProvider.ProjectID, "") != "project-uuid" || mg.GetResourceVersion() != stored.GetResourceVersion() {
		t.Error("the reconciled object must be the one the API server stored")
	}

	// A reference that does not resolve is left for crossplane-runtime's resolver to report.
	orphan := stored.DeepCopy()
	orphan.Spec.ForProvider.ChannelAssignments = append(orphan.Spec.ForProvider.ChannelAssignments,
		alertv1alpha1.ChannelAssignmentsParameters{ChannelIDRef: &xpv1.Reference{Name: "missing"}})
	if err := initialize.Initialize(context.Background(), orphan); err != nil || updates != 1 {
		t.Errorf("unresolvable reference: error %v, updates %d; want no error and no update", err, updates)
	}
}
