package config

import (
	"testing"

	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
)

func TestChannelConfigRemainsConfigurable(t *testing.T) {
	for name, provider := range map[string]*ujconfig.Provider{
		"cluster":    GetProvider(),
		"namespaced": GetProviderNamespaced(),
	} {
		t.Run(name, func(t *testing.T) {
			resource := provider.Resources["logfire_channel"]
			if resource == nil {
				t.Fatal("logfire_channel resource config is missing")
			}

			schema := resource.TerraformResource.Schema["config"]
			if schema == nil {
				t.Fatal("channel config schema is missing")
			}
			if schema.Computed {
				t.Fatal("channel config must not be observation-only")
			}
			if schema.Optional {
				t.Fatal("channel config must remain required")
			}
			if !schema.Required {
				t.Fatal("channel config must be marked required")
			}
			if schema.MinItems != 1 || schema.MaxItems != 1 {
				t.Fatalf("channel config must remain a singleton block, got min=%d max=%d", schema.MinItems, schema.MaxItems)
			}

			if !resource.SchemaElementOptions.EmbeddedObject("config") {
				t.Fatal("channel config must be generated as an embedded object")
			}

			tfPaths := resource.TFListConversionPaths()
			if len(tfPaths) != 1 || tfPaths[0] != "config" {
				t.Fatalf("unexpected Terraform list conversion paths: %v", tfPaths)
			}

			crdPaths := resource.CRDListConversionPaths()
			if len(crdPaths) != 1 || crdPaths[0] != "config" {
				t.Fatalf("unexpected CRD list conversion paths: %v", crdPaths)
			}

			fieldPaths := resource.Sensitive.GetFieldPaths()
			if _, ok := fieldPaths["config[*].auth_key"]; ok {
				t.Fatalf("stale wildcard auth_key path should have been removed: %v", fieldPaths)
			}
			if got := fieldPaths["config.auth_key"]; got != "config.authKeySecretRef" {
				t.Fatalf("unexpected auth_key sensitive path mapping: %q", got)
			}
		})
	}
}
