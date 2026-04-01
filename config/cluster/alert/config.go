package alert

import "github.com/crossplane/upjet/v2/pkg/config"

// Configure configures the cluster-scoped Alert resource.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("logfire_alert", func(r *config.Resource) {
		r.ShortGroup = "alert"
		r.Kind = "Alert"
		if r.References == nil {
			r.References = make(map[string]config.Reference)
		}
		r.References["project_id"] = config.Reference{
			TerraformName: "logfire_project",
		}
		r.References["channel_ids"] = config.Reference{
			TerraformName:     "logfire_channel",
			RefFieldName:      "ChannelIDRefs",
			SelectorFieldName: "ChannelIDSelector",
		}
	})
}
