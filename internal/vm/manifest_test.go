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
  ports:
    - host_port: 0
      container_port: 31337
      protocol: tcp
  containers:
    - name: app
      image: nginx:latest
      resource:
        cpu: 50m
        memory: 64Mi
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
		{name: "missing containers", spec: strings.Replace(validManifest, "  containers:\n    - name: app\n      image: nginx:latest\n      resource:\n        cpu: 50m\n        memory: 64Mi\n", "", 1), id: "vm-1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := RenderManifestWithID(tc.spec, tc.id); !errors.Is(err, ErrInvalid) {
				t.Fatalf("expected ErrInvalid, got %v", err)
			}
		})
	}
}
