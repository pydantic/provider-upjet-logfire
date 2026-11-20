package gateway

import "github.com/crossplane/upjet/v2/pkg/config"

// Configure configures the namespaced Gateway resources.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("logfire_gateway_endpoint", func(r *config.Resource) {
		r.ShortGroup = "gateway"
		r.Kind = "Endpoint"
	})
	p.AddResourceConfigurator("logfire_gateway_spending_limits", func(r *config.Resource) {
		r.ShortGroup = "gateway"
		r.Kind = "SpendingLimits"
	})
	p.AddResourceConfigurator("logfire_gateway_project_spending_limits", func(r *config.Resource) {
		r.ShortGroup = "gateway"
		r.Kind = "ProjectSpendingLimits"
		if r.References == nil {
			r.References = make(map[string]config.Reference)
		}
		r.References["project_id"] = config.Reference{
			TerraformName: "logfire_project",
		}
	})
}
