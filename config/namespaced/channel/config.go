package channel

import "github.com/crossplane/upjet/v2/pkg/config"

// Configure configures the namespaced Channel resource.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("logfire_channel", func(r *config.Resource) {
		r.ShortGroup = "channel"
		r.Kind = "Channel"

		// Upjet converts Terraform's single nested block into a max-items-one
		// list and incorrectly leaves it marked as computed. That makes the
		// required channel config block observation-only unless we correct it
		// before type generation.
		if schema := r.TerraformResource.Schema["config"]; schema != nil {
			schema.Computed = false
			schema.Optional = false
			schema.Required = true
			schema.MinItems = 1
			schema.MaxItems = 1
		}
		r.AddSingletonListConversion("config", "config")
	})
}
