package dashboard

import "github.com/crossplane/upjet/v2/pkg/config"

// Configure configures the cluster-scoped Dashboard resource.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("logfire_dashboard", func(r *config.Resource) {
		r.ShortGroup = "dashboard"
		r.Kind = "Dashboard"
		if r.References == nil {
			r.References = make(map[string]config.Reference)
		}
		r.References["project_id"] = config.Reference{
			TerraformName: "logfire_project",
		}
	})
}
