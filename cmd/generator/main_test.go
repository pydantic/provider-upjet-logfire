package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pydantic/provider-upjet-logfire/config"
)

func TestApplyExampleConversions(t *testing.T) {
	root := t.TempDir()
	clusterDir := filepath.Join(root, "examples-generated", "cluster", "channel", "v1alpha1")
	namespacedDir := filepath.Join(root, "examples-generated", "namespaced", "channel", "v1alpha1")
	hackDir := filepath.Join(root, "hack")
	for _, dir := range []string{clusterDir, namespacedDir, hackDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(hackDir, "boilerplate.yaml.txt"), []byte(""), 0o600); err != nil {
		t.Fatalf("failed to write boilerplate header: %v", err)
	}

	clusterManifest := []byte(`apiVersion: channel.logfire.pydantic.dev/v1alpha1
kind: Channel
metadata:
  name: example
spec:
  forProvider:
    config:
    - format: auto
      type: webhook
      url: https://example.com/logfire-webhook
    name: alerts-webhook
`)
	clusterPath := filepath.Join(clusterDir, "channel.yaml")
	if err := os.WriteFile(clusterPath, clusterManifest, 0o600); err != nil {
		t.Fatalf("failed to write cluster manifest: %v", err)
	}

	namespacedManifest := []byte(`apiVersion: channel.logfire.m.pydantic.dev/v1alpha1
kind: Channel
metadata:
  name: example
  namespace: crossplane-system
spec:
  forProvider:
    config:
    - format: auto
      type: webhook
      url: https://example.com/logfire-webhook
    name: alerts-webhook
`)
	namespacedPath := filepath.Join(namespacedDir, "channel.yaml")
	if err := os.WriteFile(namespacedPath, namespacedManifest, 0o600); err != nil {
		t.Fatalf("failed to write namespaced manifest: %v", err)
	}

	if err := applyExampleConversions(root, config.GetProvider(), config.GetProviderNamespaced()); err != nil {
		t.Fatalf("applyExampleConversions() failed: %v", err)
	}

	for name, path := range map[string]string{
		"cluster":    clusterPath,
		"namespaced": namespacedPath,
	} {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s manifest: %v", name, err)
		}
		if strings.Contains(string(content), "config:\n    -") {
			t.Fatalf("%s example still uses a singleton list: %s", name, content)
		}
		if !strings.Contains(string(content), "config:\n      format: auto") {
			t.Fatalf("%s example was not converted to an embedded object: %s", name, content)
		}
	}
}

func TestFixGeneratedChannelConnectionDetails(t *testing.T) {
	root := t.TempDir()
	for _, scope := range []string{"cluster", "namespaced"} {
		dir := filepath.Join(root, "apis", scope, "channel", "v1alpha1")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create %s: %v", dir, err)
		}
		path := filepath.Join(dir, "zz_channel_terraformed.go")
		content := `func (tr *Channel) GetConnectionDetailsMapping() map[string]string {
	return map[string]string{"config.auth_key": "config.authKeySecretRef", "config[*].auth_key": "config.authKeySecretRef"}
}
`
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("failed to write %s: %v", path, err)
		}
	}

	if err := fixGeneratedChannelConnectionDetails(root); err != nil {
		t.Fatalf("fixGeneratedChannelConnectionDetails() failed: %v", err)
	}

	for _, scope := range []string{"cluster", "namespaced"} {
		path := filepath.Join(root, "apis", scope, "channel", "v1alpha1", "zz_channel_terraformed.go")
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}
		got := string(content)
		if strings.Contains(got, `config[*].auth_key`) {
			t.Fatalf("%s still contains stale wildcard auth_key mapping: %s", scope, got)
		}
		if !strings.Contains(got, `config.auth_key`) {
			t.Fatalf("%s lost the object auth_key mapping: %s", scope, got)
		}
	}
}

func TestFixGeneratedDashboardExamples(t *testing.T) {
	root := t.TempDir()
	for _, scope := range []string{"cluster", "namespaced"} {
		dir := filepath.Join(root, "examples-generated", scope, "dashboard", "v1alpha1")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create %s: %v", dir, err)
		}
		path := filepath.Join(dir, "dashboard.yaml")
		content := `apiVersion: dashboard.logfire.pydantic.dev/v1alpha1
kind: Dashboard
spec:
  forProvider:
    definition: ${file("${path.module}/dashboard.json")}
    name: example-dashboard
    slug: example-dashboard
`
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("failed to write %s: %v", path, err)
		}
	}

	if err := fixGeneratedDashboardExamples(root); err != nil {
		t.Fatalf("fixGeneratedDashboardExamples() failed: %v", err)
	}

	for _, scope := range []string{"cluster", "namespaced"} {
		path := filepath.Join(root, "examples-generated", scope, "dashboard", "v1alpha1", "dashboard.yaml")
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}
		got := string(content)
		if strings.Contains(got, `${file("${path.module}/dashboard.json")}`) {
			t.Fatalf("%s example still contains Terraform file interpolation: %s", scope, got)
		}
		if !strings.Contains(got, `"kind": "Dashboard"`) {
			t.Fatalf("%s example did not get an inline dashboard definition: %s", scope, got)
		}
	}
}

func TestNormalizeGeneratedExamples(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "examples-generated", "cluster", "project", "v1alpha1", "project.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create project example dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("\n\napiVersion: project.logfire.pydantic.dev/v1alpha1\nkind: Project\n"), 0o600); err != nil {
		t.Fatalf("failed to write project example: %v", err)
	}

	if err := normalizeGeneratedExamples(root); err != nil {
		t.Fatalf("normalizeGeneratedExamples() failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read normalized example: %v", err)
	}
	if strings.HasPrefix(string(content), "\n") {
		t.Fatalf("expected normalized example to drop leading blank lines: %q", string(content))
	}
}
