package vm

import (
	"errors"
	"strings"
	"testing"
)

const validManifest = `apiVersion: sandboxd.o/v1
kind: Sandbox
id: placeholder
spec:
  egress: true
  ttl_seconds: 3600
  readiness_probe:
    protocol: http
    path: /
    port: 31337
    initial_delay_seconds: 10
    period_seconds: 5
    timeout_seconds: 1
    success_threshold: 1
    failure_threshold: 5
  volumes:
    - name: runtime-state
      ephemeral_storage: 128Mi
  ports:
    - host_port: 0
      container_port: 31337
      protocol: tcp
  containers:
    - name: app
      image: nginx:latest
      volume_mounts:
        - name: runtime-state
          mount_path: /srv/runtime
      resource:
        cpu: 50m
        memory: 64Mi
        ephemeral_storage: 96Mi
`

func TestRenderManifestWithID(t *testing.T) {
	rendered, req, err := RenderManifestWithID(validManifest, "vm-1")
	if err != nil {
		t.Fatalf("RenderManifestWithID: %v", err)
	}

	if req.ID != "vm-1" {
		t.Fatalf("expected req.ID vm-1, got %q", req.ID)
	}

	if len(req.Spec.Containers) != 1 {
		t.Fatalf("expected one container, got %d", len(req.Spec.Containers))
	}
	if len(req.Spec.Volumes) != 1 || req.Spec.Volumes[0].Name != "runtime-state" {
		t.Fatalf("expected one shared volume, got %+v", req.Spec.Volumes)
	}

	if req.Spec.ReadinessProbe == nil || req.Spec.ReadinessProbe.Protocol != "http" || req.Spec.ReadinessProbe.Path != "/" || req.Spec.ReadinessProbe.Port != 31337 {
		t.Fatalf("expected readiness probe to round-trip, got %+v", req.Spec.ReadinessProbe)
	}

	if req.Spec.Containers[0].Resource.EphemeralStorage != "96Mi" {
		t.Fatalf("expected container ephemeral storage to round-trip, got %+v", req.Spec.Containers[0].Resource)
	}

	if len(req.Spec.Containers[0].VolumeMounts) != 1 || req.Spec.Containers[0].VolumeMounts[0].MountPath != "/srv/runtime" {
		t.Fatalf("expected volume mount to round-trip, got %+v", req.Spec.Containers[0].VolumeMounts)
	}

	if !strings.Contains(string(rendered), "id: vm-1") {
		t.Fatalf("rendered yaml should include rewritten id, got:\n%s", string(rendered))
	}
}

func TestRenderManifestWithIDErrors(t *testing.T) {
	tests := []struct {
		name string
		spec string
		id   string
	}{
		{name: "invalid yaml", spec: "::: bad :::", id: "vm-1"},
		{name: "invalid kind", spec: strings.Replace(validManifest, "kind: Sandbox", "kind: Invalid", 1), id: "vm-1"},
		{name: "empty id", spec: validManifest, id: "   "},
		{name: "missing containers", spec: strings.Replace(validManifest, "  containers:\n    - name: app\n      image: nginx:latest\n      volume_mounts:\n        - name: runtime-state\n          mount_path: /srv/runtime\n      resource:\n        cpu: 50m\n        memory: 64Mi\n        ephemeral_storage: 96Mi\n", "", 1), id: "vm-1"},
		{name: "duplicate volume names", spec: strings.Replace(validManifest, "  volumes:\n    - name: runtime-state\n      ephemeral_storage: 128Mi\n", "  volumes:\n    - name: runtime-state\n      ephemeral_storage: 128Mi\n    - name: runtime-state\n      ephemeral_storage: 64Mi\n", 1), id: "vm-1"},
		{name: "missing volume storage", spec: strings.Replace(validManifest, "      ephemeral_storage: 128Mi\n", "", 1), id: "vm-1"},
		{name: "unknown volume mount", spec: strings.Replace(validManifest, "        - name: runtime-state\n", "        - name: missing-volume\n", 1), id: "vm-1"},
		{name: "missing mount path", spec: strings.Replace(validManifest, "          mount_path: /srv/runtime\n", "", 1), id: "vm-1"},
		{name: "relative mount path", spec: strings.Replace(validManifest, "          mount_path: /srv/runtime\n", "          mount_path: srv/runtime\n", 1), id: "vm-1"},
		{name: "tmp mount path", spec: strings.Replace(validManifest, "          mount_path: /srv/runtime\n", "          mount_path: /tmp\n", 1), id: "vm-1"},
		{name: "http readiness probe requires path", spec: strings.Replace(validManifest, "    path: /\n", "", 1), id: "vm-1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := RenderManifestWithID(tc.spec, tc.id); !errors.Is(err, ErrInvalid) {
				t.Fatalf("expected ErrInvalid, got %v", err)
			}
		})
	}
}
