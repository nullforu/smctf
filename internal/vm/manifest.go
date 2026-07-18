package vm

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type SandboxManifest struct {
	APIVersion string         `yaml:"apiVersion"`
	Kind       string         `yaml:"kind"`
	ID         string         `yaml:"id"`
	Spec       map[string]any `yaml:"spec"`
}

func RenderManifestWithID(rawSpec string, id string) ([]byte, CreateSandboxRequest, error) {
	var manifest SandboxManifest
	if err := yaml.Unmarshal([]byte(rawSpec), &manifest); err != nil {
		return nil, CreateSandboxRequest{}, fmt.Errorf("%w: parse yaml", ErrInvalid)
	}

	if !strings.EqualFold(strings.TrimSpace(manifest.Kind), "Sandbox") {
		return nil, CreateSandboxRequest{}, fmt.Errorf("%w: kind must be Sandbox", ErrInvalid)
	}

	if strings.TrimSpace(id) == "" {
		return nil, CreateSandboxRequest{}, fmt.Errorf("%w: id is required", ErrInvalid)
	}

	if err := validateSandboxSpec(manifest.Spec); err != nil {
		return nil, CreateSandboxRequest{}, err
	}

	manifest.ID = strings.TrimSpace(id)
	rendered, err := yaml.Marshal(&manifest)
	if err != nil {
		return nil, CreateSandboxRequest{}, fmt.Errorf("%w: render yaml", ErrInvalid)
	}

	var req CreateSandboxRequest
	if err := yaml.Unmarshal(rendered, &req); err != nil {
		return nil, CreateSandboxRequest{}, fmt.Errorf("%w: parse rendered yaml", ErrInvalid)
	}

	return rendered, req, nil
}

func validateSandboxSpec(spec map[string]any) error {
	if len(spec) == 0 {
		return fmt.Errorf("%w: spec is required", ErrInvalid)
	}

	containers, ok := spec["containers"]
	if !ok {
		return fmt.Errorf("%w: spec.containers is required", ErrInvalid)
	}

	containerList, ok := containers.([]any)
	if !ok || len(containerList) == 0 {
		return fmt.Errorf("%w: spec.containers is required", ErrInvalid)
	}

	return nil
}
