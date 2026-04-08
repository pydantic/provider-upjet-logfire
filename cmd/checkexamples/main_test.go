package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMissingExamplesAcceptsGeneratedAndStaticExamples(t *testing.T) {
	root := t.TempDir()
	crdDir := filepath.Join(root, "crds")
	staticDir := filepath.Join(root, "examples")
	generatedDir := filepath.Join(root, "examples-generated")

	writeFile(t, filepath.Join(crdDir, "project.yaml"), `
kind: CustomResourceDefinition
spec:
  group: logfire.pydantic.dev
  names:
    kind: Project
  versions:
    - name: v1alpha1
`)
	writeFile(t, filepath.Join(crdDir, "providerconfig.yaml"), `
kind: CustomResourceDefinition
spec:
  group: logfire.pydantic.dev
  names:
    kind: ProviderConfig
  versions:
    - name: v1beta1
`)
	writeFile(t, filepath.Join(generatedDir, "project.yaml"), `
apiVersion: logfire.pydantic.dev/v1alpha1
kind: Project
`)
	writeFile(t, filepath.Join(staticDir, "providerconfig.yaml"), `
apiVersion: logfire.pydantic.dev/v1beta1
kind: ProviderConfig
`)

	missing, err := missingExamples(crdDir, []string{generatedDir, staticDir})
	if err != nil {
		t.Fatalf("missingExamples() error = %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("missingExamples() = %v, want no missing examples", missing)
	}
}

func TestMissingExamplesIgnoresExpectedExceptions(t *testing.T) {
	root := t.TempDir()
	crdDir := filepath.Join(root, "crds")
	exampleDir := filepath.Join(root, "examples")

	writeFile(t, filepath.Join(crdDir, "usage.yaml"), `
kind: CustomResourceDefinition
spec:
  group: logfire.pydantic.dev
  names:
    kind: ProviderConfigUsage
  versions:
    - name: v1beta1
`)
	writeFile(t, filepath.Join(exampleDir, "project.yaml"), `
apiVersion: logfire.pydantic.dev/v1alpha1
kind: Project
`)

	missing, err := missingExamples(crdDir, []string{exampleDir})
	if err != nil {
		t.Fatalf("missingExamples() error = %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("missingExamples() = %v, want no missing examples because ProviderConfigUsage is exempt", missing)
	}
}

func TestRunReportsMissingExamples(t *testing.T) {
	root := t.TempDir()
	crdDir := filepath.Join(root, "crds")
	exampleDir := filepath.Join(root, "examples")
	if err := os.MkdirAll(exampleDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", exampleDir, err)
	}

	writeFile(t, filepath.Join(crdDir, "dashboard.yaml"), `
kind: CustomResourceDefinition
spec:
  group: logfire.pydantic.dev
  names:
    kind: Dashboard
  versions:
    - name: v1alpha1
`)

	var out bytes.Buffer
	err := run([]string{crdDir, exampleDir}, &out)
	if err == nil {
		t.Fatal("run() error = nil, want missing example error")
	}
	if !strings.Contains(err.Error(), "Dashboard.logfire.pydantic.dev/v1alpha1") {
		t.Fatalf("run() error = %v, want missing dashboard type", err)
	}
	if out.Len() != 0 {
		t.Fatalf("run() wrote %q, want no success output", out.String())
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", path, err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(contents)+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}
