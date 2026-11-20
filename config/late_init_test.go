package config

import (
	"slices"
	"testing"

	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
	"k8s.io/utils/ptr"

	alertv1alpha1 "github.com/pydantic/provider-upjet-logfire/apis/cluster/alert/v1alpha1"
	gatewayv1alpha1 "github.com/pydantic/provider-upjet-logfire/apis/cluster/gateway/v1alpha1"
	projectv1alpha1 "github.com/pydantic/provider-upjet-logfire/apis/cluster/project/v1alpha1"
	tokenv1alpha1 "github.com/pydantic/provider-upjet-logfire/apis/cluster/token/v1alpha1"
)

func TestLateInitializationSkipsClearableFields(t *testing.T) {
	for name, provider := range map[string]*ujconfig.Provider{
		"cluster":    GetProvider(),
		"namespaced": GetProviderNamespaced(),
	} {
		t.Run(name, func(t *testing.T) {
			for resource, fields := range map[string]struct{ skipped, kept []string }{
				"logfire_project":                 {skipped: []string{"description"}, kept: []string{"visibility"}},
				"logfire_alert":                   {skipped: []string{"description", "environments", "instance_policy"}, kept: []string{"active", "channel_assignments.schedule_id"}},
				"logfire_channel":                 {skipped: []string{"config.recipients", "config.url"}, kept: []string{"active", "config.base_url"}},
				"logfire_gateway_spending_limits": {skipped: []string{"daily_usd", "monthly_usd", "weekly_usd"}},
				"logfire_gateway_endpoint":        {skipped: []string{"description"}, kept: []string{"providers.weight"}},
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

// The generated LateInitialize methods copy only what Terraform would keep when
// the spec omits it.
func TestGeneratedLateInitialization(t *testing.T) {
	late := func(t *testing.T, li func([]byte) (bool, error), observed string) {
		t.Helper()
		if _, err := li([]byte(observed)); err != nil {
			t.Fatal(err)
		}
	}

	alert := &alertv1alpha1.Alert{}
	late(t, alert.LateInitialize, `{"active":true,"description":"observed","environments":["prod"],
		"channel_assignments":[{"channel_id":"c1","schedule_id":"s1"}],
		"instance_policy":{"identity_columns":["x"],"missing_behavior":"expire","missing_runs":1,"pending_runs":1}}`)
	a := alert.Spec.ForProvider
	if a.Description != nil || a.Environments != nil || a.InstancePolicy != nil {
		t.Errorf("alert copied clearable fields: description %v, environments %v, instance policy %v",
			a.Description, a.Environments, a.InstancePolicy)
	}
	if !ptr.Deref(a.Active, false) {
		t.Error("alert must copy active, which has a Logfire default")
	}
	if len(a.ChannelAssignments) != 1 || ptr.Deref(a.ChannelAssignments[0].ScheduleID, "") != "s1" {
		t.Errorf("alert must copy channel assignments whole, got %+v", a.ChannelAssignments)
	}

	project := &projectv1alpha1.Project{}
	late(t, project.LateInitialize, `{"description":"observed","visibility":"private"}`)
	if project.Spec.ForProvider.Description != nil || ptr.Deref(project.Spec.ForProvider.Visibility, "") != "private" {
		t.Errorf("project: description %v, visibility %v; want only visibility copied",
			project.Spec.ForProvider.Description, project.Spec.ForProvider.Visibility)
	}

	limits := &gatewayv1alpha1.SpendingLimits{}
	late(t, limits.LateInitialize, `{"daily_usd":1,"weekly_usd":2,"monthly_usd":3}`)
	if l := limits.Spec.ForProvider; l.DailyUsd != nil || l.WeeklyUsd != nil || l.MonthlyUsd != nil {
		t.Errorf("spending limits copied windows: %+v", l)
	}

	token := &tokenv1alpha1.WriteToken{}
	late(t, token.LateInitialize, `{"expires_at":"2099-12-31T23:59:59Z"}`)
	if ptr.Deref(token.Spec.ForProvider.ExpiresAt, "") != "2099-12-31T23:59:59Z" {
		t.Error("write token must keep expires_at, whose change replaces the token")
	}
}
