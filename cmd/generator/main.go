package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
	exampleconversion "github.com/crossplane/upjet/v2/pkg/examples/conversion"
	"github.com/crossplane/upjet/v2/pkg/pipeline"

	"github.com/pydantic/provider-upjet-logfire/config"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] == "" {
		panic("root directory is required to be given as argument")
	}
	if err := run(os.Args[1]); err != nil {
		panic(err)
	}
}

func run(rootDir string) error {
	absRootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return fmt.Errorf("cannot calculate the absolute path with %s", rootDir)
	}
	pc := config.GetProvider()
	pns := config.GetProviderNamespaced()
	pipeline.Run(pc, pns, absRootDir)
	if err := applyExampleConversions(absRootDir, pc, pns); err != nil {
		return err
	}
	if err := fixGeneratedChannelConnectionDetails(absRootDir); err != nil {
		return err
	}
	if err := fixGeneratedDashboardExamples(absRootDir); err != nil {
		return err
	}
	return normalizeGeneratedExamples(absRootDir)
}

func applyExampleConversions(rootDir string, pc, pns *ujconfig.Provider) error {
	licenseHeaderPath := filepath.Join(rootDir, "hack", "boilerplate.yaml.txt")
	if _, err := os.Stat(licenseHeaderPath); err != nil {
		if os.IsNotExist(err) {
			licenseHeaderPath = ""
		} else {
			return fmt.Errorf("cannot inspect example manifest license header: %w", err)
		}
	}
	if err := exampleconversion.ApplyAPIConverters(pc, filepath.Join(rootDir, "examples-generated", "cluster"), licenseHeaderPath); err != nil {
		return fmt.Errorf("cannot convert singleton lists in cluster examples: %w", err)
	}
	if err := exampleconversion.ApplyAPIConverters(pns, filepath.Join(rootDir, "examples-generated", "namespaced"), licenseHeaderPath); err != nil {
		return fmt.Errorf("cannot convert singleton lists in namespaced examples: %w", err)
	}
	return nil
}

func fixGeneratedChannelConnectionDetails(rootDir string) error {
	wildcardAuthKeyPattern := regexp.MustCompile(`,\s*"config\[\*\]\.auth_key": "config\.authKeySecretRef"|"config\[\*\]\.auth_key": "config\.authKeySecretRef",\s*`)

	for _, rel := range []string{
		filepath.Join("apis", "cluster", "channel", "v1alpha1", "zz_channel_terraformed.go"),
		filepath.Join("apis", "namespaced", "channel", "v1alpha1", "zz_channel_terraformed.go"),
	} {
		path := filepath.Join(rootDir, rel)
		// #nosec G304 -- path is built from known generated file locations inside the repo root.
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("cannot read generated channel terraformed file %s: %w", path, err)
		}

		updated := wildcardAuthKeyPattern.ReplaceAllString(string(content), "")
		if updated == string(content) {
			continue
		}
		if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
			return fmt.Errorf("cannot rewrite generated channel terraformed file %s: %w", path, err)
		}
	}
	return nil
}

func fixGeneratedDashboardExamples(rootDir string) error {
	inlineDefinition := `    definition: |
      {
        "kind": "Dashboard",
        "metadata": {
          "name": "example-dashboard"
        },
        "spec": {
          "display": {
            "name": "example-dashboard",
            "description": null
          },
          "panels": {},
          "layouts": [],
          "variables": [],
          "duration": "1h",
          "refreshInterval": "0s",
          "datasources": {}
        }
      }`
	for _, rel := range []string{
		filepath.Join("examples-generated", "cluster", "dashboard", "v1alpha1", "dashboard.yaml"),
		filepath.Join("examples-generated", "namespaced", "dashboard", "v1alpha1", "dashboard.yaml"),
	} {
		path := filepath.Join(rootDir, rel)
		// #nosec G304 -- path is built from known generated file locations inside the repo root.
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("cannot read generated dashboard example %s: %w", path, err)
		}
		updated := strings.ReplaceAll(string(content), `    definition: ${file("${path.module}/dashboard.json")}`, inlineDefinition)
		if updated == string(content) {
			continue
		}
		if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
			return fmt.Errorf("cannot rewrite generated dashboard example %s: %w", path, err)
		}
	}
	return nil
}

func normalizeGeneratedExamples(rootDir string) error {
	return filepath.WalkDir(filepath.Join(rootDir, "examples-generated"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".yaml" {
			return nil
		}

		// #nosec G304 -- path is returned by walking the repo's examples-generated tree.
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("cannot read generated example %s: %w", path, err)
		}
		updated := strings.TrimLeft(string(content), "\r\n")
		if updated == string(content) {
			return nil
		}
		if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
			return fmt.Errorf("cannot rewrite generated example %s: %w", path, err)
		}
		return nil
	})
}
