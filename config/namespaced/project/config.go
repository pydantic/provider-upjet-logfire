package project

import "github.com/crossplane/upjet/v2/pkg/config"

// Configure configures the namespaced Project resource.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("logfire_project", func(r *config.Resource) {
		r.ShortGroup = "project"
	})
}
