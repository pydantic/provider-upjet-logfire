package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type crdDocument struct {
	Kind string `yaml:"kind"`
	Spec struct {
		Group string `yaml:"group"`
		Names struct {
			Kind string `yaml:"kind"`
		} `yaml:"names"`
		Versions []struct {
			Name string `yaml:"name"`
		} `yaml:"versions"`
	} `yaml:"spec"`
}

type manifestDocument struct {
	Kind       string `yaml:"kind"`
	APIVersion string `yaml:"apiVersion"`
}

var exceptionTypes = map[string]struct{}{
	"ProviderConfigUsage.logfire.pydantic.dev/v1beta1":   {},
	"ProviderConfigUsage.logfire.m.pydantic.dev/v1beta1": {},
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	if len(args) < 2 {
		return errors.New("usage: checkexamples <crd dir> <example dir> [<example dir>...]")
	}

	missing, err := missingExamples(args[0], args[1:])
	if err != nil {
		return err
	}
	if len(missing) == 0 {
		_, _ = fmt.Fprintln(stdout, "All CRDs have at least one example.")
		return nil
	}
	return fmt.Errorf("please add example manifests for the following types: %s", strings.Join(missing, ", "))
}

func missingExamples(crdDir string, exampleDirs []string) ([]string, error) {
	knownCRDTypes, err := loadCRDTypes(crdDir)
	if err != nil {
		return nil, err
	}

	exampleTypes := make(map[string]struct{})
	for _, dir := range exampleDirs {
		types, err := loadExampleTypes(dir)
		if err != nil {
			return nil, err
		}
		for typ := range types {
			exampleTypes[typ] = struct{}{}
		}
	}

	missing := make([]string, 0, len(knownCRDTypes))
	for typ := range knownCRDTypes {
		if _, ok := exampleTypes[typ]; ok {
			continue
		}
		if _, ok := exceptionTypes[typ]; ok {
			continue
		}
		missing = append(missing, typ)
	}
	sort.Strings(missing)
	return missing, nil
}

func loadCRDTypes(dir string) (map[string]struct{}, error) {
	types := make(map[string]struct{})
	err := walkYAMLDocuments(dir, func(path string, dec *yaml.Decoder) error {
		for {
			var doc crdDocument
			if err := dec.Decode(&doc); err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return fmt.Errorf("cannot decode CRD %s: %w", path, err)
			}
			if doc.Kind != "CustomResourceDefinition" || doc.Spec.Group == "" || doc.Spec.Names.Kind == "" {
				continue
			}
			for _, version := range doc.Spec.Versions {
				if version.Name == "" {
					continue
				}
				types[fmt.Sprintf("%s.%s/%s", doc.Spec.Names.Kind, doc.Spec.Group, version.Name)] = struct{}{}
			}
		}
	})
	return types, err
}

func loadExampleTypes(dir string) (map[string]struct{}, error) {
	types := make(map[string]struct{})
	err := walkYAMLDocuments(dir, func(path string, dec *yaml.Decoder) error {
		for {
			var doc manifestDocument
			if err := dec.Decode(&doc); err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return fmt.Errorf("cannot decode example %s: %w", path, err)
			}
			if doc.Kind == "" || doc.APIVersion == "" {
				continue
			}
			types[fmt.Sprintf("%s.%s", doc.Kind, doc.APIVersion)] = struct{}{}
		}
	})
	return types, err
}

func walkYAMLDocuments(root string, fn func(path string, dec *yaml.Decoder) error) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".yaml" {
			return nil
		}

		// #nosec G304 -- path is returned by walking the caller-provided manifest directory.
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("cannot open %s: %w", path, err)
		}

		dec := yaml.NewDecoder(f)
		if err := fn(path, dec); err != nil {
			if closeErr := f.Close(); closeErr != nil {
				return fmt.Errorf("cannot close %s after error: %w", path, closeErr)
			}
			return err
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("cannot close %s: %w", path, err)
		}
		return nil
	})
}
