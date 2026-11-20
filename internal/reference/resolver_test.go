package reference

import (
	"context"
	"strings"
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/pydantic/provider-upjet-logfire/apis/cluster"
	alertv1alpha1 "github.com/pydantic/provider-upjet-logfire/apis/cluster/alert/v1alpha1"
	channelv1alpha1 "github.com/pydantic/provider-upjet-logfire/apis/cluster/channel/v1alpha1"
	projectv1alpha1 "github.com/pydantic/provider-upjet-logfire/apis/cluster/project/v1alpha1"
)

type recorder struct {
	t       *testing.T
	patches []string
}

func setup(t *testing.T, alert *alertv1alpha1.Alert) (client.Client, client.Client, *recorder) {
	t.Helper()
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
	alert.SetName("alert")
	store := fake.NewClientBuilder().WithScheme(scheme).WithObjects(project, channel, alert).Build()
	rec := &recorder{t: t}
	c := interceptor.NewClient(store, interceptor.Funcs{
		Patch: func(ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
			if patch.Type() != types.MergePatchType {
				t.Errorf("patch type %s, want a JSON merge patch", patch.Type())
			}
			data, err := patch.Data(obj)
			if err != nil {
				t.Fatal(err)
			}
			rec.patches = append(rec.patches, string(data))
			return c.Patch(ctx, obj, patch, opts...)
		},
		Update: func(context.Context, client.WithWatch, client.Object, ...client.UpdateOption) error {
			t.Error("the resolver must patch, not update")
			return nil
		},
	})
	return store, c, rec
}

func load(t *testing.T, c client.Client) *alertv1alpha1.Alert {
	t.Helper()
	a := &alertv1alpha1.Alert{}
	if err := c.Get(context.Background(), client.ObjectKey{Name: "alert"}, a); err != nil {
		t.Fatal(err)
	}
	return a
}

func TestResolverPatchesResolvedFieldsOnly(t *testing.T) {
	alert := &alertv1alpha1.Alert{}
	alert.Spec.ForProvider.ProjectIDRef = &xpv1.Reference{Name: "proj"}
	alert.Spec.ForProvider.ChannelAssignments = []alertv1alpha1.ChannelAssignmentsParameters{{ChannelIDRef: &xpv1.Reference{Name: "hook"}}}
	alert.Spec.ForProvider.Description = ptr.To("kept")
	store, c, rec := setup(t, alert)
	resolve := NewResolver(c)

	mg := load(t, store)
	for range 2 {
		if err := resolve.ResolveReferences(context.Background(), mg); err != nil {
			t.Fatal(err)
		}
	}
	if len(rec.patches) != 1 {
		t.Fatalf("patches = %v, want one: the second pass has nothing to resolve", rec.patches)
	}
	if !strings.Contains(rec.patches[0], `"resourceVersion"`) {
		t.Errorf("patch %s must carry the resource version", rec.patches[0])
	}
	stored := load(t, store)
	p := stored.Spec.ForProvider
	if ptr.Deref(p.ProjectID, "") != "project-uuid" || len(p.ChannelAssignments) != 1 ||
		ptr.Deref(p.ChannelAssignments[0].ChannelID, "") != "channel-uuid" || ptr.Deref(p.Description, "") != "kept" {
		t.Errorf("stored spec = %+v, want both references resolved and other fields kept", p)
	}
	if mg.GetResourceVersion() != stored.GetResourceVersion() {
		t.Error("the reconciled object must be the one the API server stored")
	}
}

// An Alert that the default resolver left without projectId gets it back, and
// its channel assignments stay.
func TestResolverRepairsAlertMissingProjectID(t *testing.T) {
	alert := &alertv1alpha1.Alert{}
	alert.Spec.ForProvider.ProjectIDRef = &xpv1.Reference{Name: "proj"}
	alert.Spec.ForProvider.ChannelAssignments = []alertv1alpha1.ChannelAssignmentsParameters{
		{ChannelID: ptr.To("channel-uuid"), ChannelIDRef: &xpv1.Reference{Name: "hook"}},
	}
	store, c, rec := setup(t, alert)
	if err := NewResolver(c).ResolveReferences(context.Background(), load(t, store)); err != nil {
		t.Fatal(err)
	}
	if len(rec.patches) != 1 || strings.Contains(rec.patches[0], "channelAssignments") {
		t.Errorf("patches = %v, want one naming only projectId", rec.patches)
	}
	p := load(t, store).Spec.ForProvider
	if ptr.Deref(p.ProjectID, "") != "project-uuid" || len(p.ChannelAssignments) != 1 {
		t.Errorf("stored spec = %+v, want projectId restored and channel assignments kept", p)
	}
}

// A reference that does not resolve yet, such as a Channel not created yet,
// fails the reconcile and saves nothing.
func TestResolverSavesNothingWhenAReferenceFails(t *testing.T) {
	alert := &alertv1alpha1.Alert{}
	alert.Spec.ForProvider.ProjectIDRef = &xpv1.Reference{Name: "proj"}
	alert.Spec.ForProvider.ChannelAssignments = []alertv1alpha1.ChannelAssignmentsParameters{{ChannelIDRef: &xpv1.Reference{Name: "missing"}}}
	store, c, rec := setup(t, alert)
	if err := NewResolver(c).ResolveReferences(context.Background(), load(t, store)); err == nil {
		t.Error("want an error for the missing Channel")
	}
	if len(rec.patches) != 0 || load(t, store).Spec.ForProvider.ProjectID != nil {
		t.Errorf("patches = %v, want nothing saved", rec.patches)
	}
}

// A resolver working from a stale copy must not overwrite a newer edit. The
// conflict fails this reconcile, and the next one starts from the new object.
func TestResolverDoesNotOverwriteNewerEdits(t *testing.T) {
	alert := &alertv1alpha1.Alert{}
	alert.Spec.ForProvider.ProjectIDRef = &xpv1.Reference{Name: "proj"}
	alert.Spec.ForProvider.ChannelAssignments = []alertv1alpha1.ChannelAssignmentsParameters{{ChannelIDRef: &xpv1.Reference{Name: "hook"}}}
	store, c, _ := setup(t, alert)
	stale := load(t, store)
	newer := load(t, store)
	newer.Spec.ForProvider.ChannelAssignments[0].ScheduleID = ptr.To("schedule")
	if err := store.Update(context.Background(), newer); err != nil {
		t.Fatal(err)
	}

	resolve := NewResolver(c)
	err := resolve.ResolveReferences(context.Background(), stale)
	if !kerrors.IsConflict(errors.Cause(err)) {
		t.Fatalf("err = %v, want a conflict", err)
	}
	if got := load(t, store).Spec.ForProvider; got.ProjectID != nil || ptr.Deref(got.ChannelAssignments[0].ScheduleID, "") != "schedule" {
		t.Fatalf("stored spec = %+v, want the newer edit untouched", got)
	}

	if err := resolve.ResolveReferences(context.Background(), load(t, store)); err != nil {
		t.Fatal(err)
	}
	got := load(t, store).Spec.ForProvider
	if ptr.Deref(got.ProjectID, "") != "project-uuid" || ptr.Deref(got.ChannelAssignments[0].ChannelID, "") != "channel-uuid" ||
		ptr.Deref(got.ChannelAssignments[0].ScheduleID, "") != "schedule" {
		t.Errorf("stored spec = %+v, want references resolved and the schedule kept", got)
	}
}
