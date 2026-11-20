package config

import (
	"sort"

	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// replacedOnChange lists, per resource, the optional fields whose change makes
// Terraform replace the resource. Upjet refuses replacements, so these keep
// late-initialization: a field removed from the spec keeps its current value
// instead of planning a replacement that fails.
var replacedOnChange = map[string]map[string]bool{
	"logfire_read_token":  {"expires_at": true},
	"logfire_write_token": {"expires_at": true},
}

// LateInitializeComputedOnly stops late-initialization from copying optional
// fields that have no server-side default into the spec. Otherwise a field
// removed from the spec is copied back from the observed resource and never
// cleared, which differs from Terraform, where removing it clears it. Computed
// fields still late-initialize, because Terraform keeps their value as well.
func LateInitializeComputedOnly() ujconfig.ResourceOption {
	return func(r *ujconfig.Resource) {
		keep := replacedOnChange[r.Name]
		var ignored []string
		var walk func(prefix string, fields map[string]*schema.Schema)
		walk = func(prefix string, fields map[string]*schema.Schema) {
			for name, s := range fields {
				path := prefix + name
				if s.Optional && !s.Computed && !keep[path] {
					ignored = append(ignored, path)
					continue
				}
				// Late-initialization fills a collection only when the spec omits
				// it, and then copies it whole. Skipping a field inside a set or a
				// list would copy elements without it, and Upjet would then ignore
				// changes at an element index, which Terraform rejects for sets. So
				// only a single nested block, such as a channel's config, is walked.
				if nested, ok := s.Elem.(*schema.Resource); ok && s.Type == schema.TypeList && s.MaxItems == 1 {
					walk(path+".", nested.Schema)
				}
			}
		}
		walk("", r.TerraformResource.Schema)
		sort.Strings(ignored)
		r.LateInitializer.IgnoredFields = append(r.LateInitializer.IgnoredFields, ignored...)
	}
}
