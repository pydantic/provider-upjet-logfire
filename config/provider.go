package config

import (
	// Note(turkenh): we are importing this to embed provider schema document
	_ "embed"

	ujconfig "github.com/crossplane/upjet/v2/pkg/config"

	alertCluster "github.com/pydantic/provider-upjet-logfire/config/cluster/alert"
	channelCluster "github.com/pydantic/provider-upjet-logfire/config/cluster/channel"
	dashboardCluster "github.com/pydantic/provider-upjet-logfire/config/cluster/dashboard"
	projectCluster "github.com/pydantic/provider-upjet-logfire/config/cluster/project"
	readTokenCluster "github.com/pydantic/provider-upjet-logfire/config/cluster/readtoken"
	tokenCluster "github.com/pydantic/provider-upjet-logfire/config/cluster/token"
	alertNamespaced "github.com/pydantic/provider-upjet-logfire/config/namespaced/alert"
	channelNamespaced "github.com/pydantic/provider-upjet-logfire/config/namespaced/channel"
	dashboardNamespaced "github.com/pydantic/provider-upjet-logfire/config/namespaced/dashboard"
	projectNamespaced "github.com/pydantic/provider-upjet-logfire/config/namespaced/project"
	readTokenNamespaced "github.com/pydantic/provider-upjet-logfire/config/namespaced/readtoken"
	tokenNamespaced "github.com/pydantic/provider-upjet-logfire/config/namespaced/token"
)

const (
	resourcePrefix = "logfire"
	modulePath     = "github.com/pydantic/provider-upjet-logfire"
)

//go:embed schema.json
var providerSchema string

//go:embed provider-metadata.yaml
var providerMetadata string

// GetProvider returns provider configuration
func GetProvider() *ujconfig.Provider {
	pc := ujconfig.NewProvider([]byte(providerSchema), resourcePrefix, modulePath, []byte(providerMetadata),
		ujconfig.WithRootGroup("logfire.pydantic.dev"),
		ujconfig.WithIncludeList(ExternalNameConfigured()),
		ujconfig.WithFeaturesPackage("internal/features"),
		ujconfig.WithDefaultResourceOptions(
			ExternalNameConfigurations(),
		))

	for _, configure := range []func(provider *ujconfig.Provider){
		// add custom config functions
		alertCluster.Configure,
		channelCluster.Configure,
		dashboardCluster.Configure,
		projectCluster.Configure,
		readTokenCluster.Configure,
		tokenCluster.Configure,
	} {
		configure(pc)
	}

	pc.ConfigureResources()
	return pc
}

// GetProviderNamespaced returns the namespaced provider configuration
func GetProviderNamespaced() *ujconfig.Provider {
	pc := ujconfig.NewProvider([]byte(providerSchema), resourcePrefix, modulePath, []byte(providerMetadata),
		ujconfig.WithRootGroup("logfire.m.pydantic.dev"),
		ujconfig.WithIncludeList(ExternalNameConfigured()),
		ujconfig.WithFeaturesPackage("internal/features"),
		ujconfig.WithDefaultResourceOptions(
			ExternalNameConfigurations(),
		),
		ujconfig.WithExampleManifestConfiguration(ujconfig.ExampleManifestConfiguration{
			ManagedResourceNamespace: "crossplane-system",
		}))

	for _, configure := range []func(provider *ujconfig.Provider){
		// add custom config functions
		alertNamespaced.Configure,
		channelNamespaced.Configure,
		dashboardNamespaced.Configure,
		projectNamespaced.Configure,
		readTokenNamespaced.Configure,
		tokenNamespaced.Configure,
	} {
		configure(pc)
	}

	pc.ConfigureResources()
	return pc
}
