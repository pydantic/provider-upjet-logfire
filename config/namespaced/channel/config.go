package channel

import "github.com/crossplane/upjet/v2/pkg/config"

// Configure configures the namespaced Channel resource.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("logfire_channel", func(r *config.Resource) {
		r.ShortGroup = "channel"
		r.Kind = "Channel"
	})
}
