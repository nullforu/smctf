package vm

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type SandboxManifest struct {
	APIVersion string      `yaml:"apiVersion"`
	Kind       string      `yaml:"kind"`
	ID         string      `yaml:"id"`
	Spec       SandboxSpec `yaml:"spec"`
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

	if len(manifest.Spec.Containers) == 0 {
		return nil, CreateSandboxRequest{}, fmt.Errorf("%w: spec.containers is required", ErrInvalid)
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

func validateSandboxSpec(spec SandboxSpec) error {
	if spec.ReadinessProbe != nil {
		protocol := strings.TrimSpace(spec.ReadinessProbe.Protocol)
		if strings.EqualFold(protocol, "http") && strings.TrimSpace(spec.ReadinessProbe.Path) == "" {
			return fmt.Errorf("%w: spec.readiness_probe.path is required for http protocol", ErrInvalid)
		}
	}

	volumes := make(map[string]struct{}, len(spec.Volumes))
	for _, volume := range spec.Volumes {
		name := strings.TrimSpace(volume.Name)
		if name == "" {
			return fmt.Errorf("%w: spec.volumes.name is required", ErrInvalid)
		}

		if _, exists := volumes[name]; exists {
			return fmt.Errorf("%w: spec.volumes.name must be unique", ErrInvalid)
		}

		if strings.TrimSpace(volume.EphemeralStorage) == "" {
			return fmt.Errorf("%w: spec.volumes.ephemeral_storage is required", ErrInvalid)
		}

		volumes[name] = struct{}{}
	}

	for _, container := range spec.Containers {
		for _, mount := range container.VolumeMounts {
			name := strings.TrimSpace(mount.Name)
			if name == "" {
				return fmt.Errorf("%w: spec.containers.volume_mounts.name is required", ErrInvalid)
			}

			if _, exists := volumes[name]; !exists {
				return fmt.Errorf("%w: spec.containers.volume_mounts.name must reference spec.volumes", ErrInvalid)
			}

			mountPath := strings.TrimSpace(mount.MountPath)
			if mountPath == "" {
				return fmt.Errorf("%w: spec.containers.volume_mounts.mount_path is required", ErrInvalid)
			}

			if !strings.HasPrefix(mountPath, "/") {
				return fmt.Errorf("%w: spec.containers.volume_mounts.mount_path must be absolute", ErrInvalid)
			}

			if mountPath == "/tmp" {
				return fmt.Errorf("%w: spec.containers.volume_mounts.mount_path cannot be /tmp", ErrInvalid)
			}
		}
	}

	return nil
}
