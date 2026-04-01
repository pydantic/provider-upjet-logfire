package readtoken

import "github.com/crossplane/upjet/v2/pkg/config"

// Configure configures the namespaced ReadToken resource.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("logfire_read_token", func(r *config.Resource) {
		r.ShortGroup = "token"
		r.Kind = "ReadToken"
		if r.References == nil {
			r.References = make(map[string]config.Reference)
		}
		r.References["project_id"] = config.Reference{
			TerraformName: "logfire_project",
		}
	})
}
