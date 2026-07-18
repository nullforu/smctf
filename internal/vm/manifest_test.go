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
      cap_add:
        - SYS_PTRACE
      cap_drop:
        - ALL
      security_opt:
        - no-new-privileges:true
        - seccomp=unconfined
      read_only: true
      tmpfs:
        - mount_path: /tmp
          options: rw,nosuid,nodev,exec,mode=1777
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

	containers, ok := req.Spec["containers"].([]any)
	if !ok || len(containers) != 1 {
		t.Fatalf("expected one container, got %#v", req.Spec["containers"])
	}

	container, ok := containers[0].(map[string]any)
	if !ok {
		t.Fatalf("expected container map, got %#v", containers[0])
	}

	securityOpt, ok := container["security_opt"].([]any)
	if !ok || len(securityOpt) != 2 {
		t.Fatalf("expected security_opt to be preserved, got %#v", container["security_opt"])
	}

	tmpfs, ok := container["tmpfs"].([]any)
	if !ok || len(tmpfs) != 1 {
		t.Fatalf("expected tmpfs to be preserved, got %#v", container["tmpfs"])
	}

	if !strings.Contains(string(rendered), "id: vm-1") {
		t.Fatalf("rendered yaml should include rewritten id, got:\n%s", string(rendered))
	}

	if !strings.Contains(string(rendered), "security_opt:") || !strings.Contains(string(rendered), "tmpfs:") {
		t.Fatalf("rendered yaml should preserve custom fields, got:\n%s", string(rendered))
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
		{name: "missing spec", spec: "apiVersion: sandboxd.o/v1\nkind: Sandbox\nid: placeholder\n", id: "vm-1"},
		{name: "missing containers", spec: strings.Replace(validManifest, "  containers:\n    - name: app\n      image: nginx:latest\n      cap_add:\n        - SYS_PTRACE\n      cap_drop:\n        - ALL\n      security_opt:\n        - no-new-privileges:true\n        - seccomp=unconfined\n      read_only: true\n      tmpfs:\n        - mount_path: /tmp\n          options: rw,nosuid,nodev,exec,mode=1777\n      resource:\n        cpu: 50m\n        memory: 64Mi\n", "", 1), id: "vm-1"},
		{name: "empty containers", spec: strings.Replace(validManifest, "  containers:\n    - name: app\n      image: nginx:latest\n      cap_add:\n        - SYS_PTRACE\n      cap_drop:\n        - ALL\n      security_opt:\n        - no-new-privileges:true\n        - seccomp=unconfined\n      read_only: true\n      tmpfs:\n        - mount_path: /tmp\n          options: rw,nosuid,nodev,exec,mode=1777\n      resource:\n        cpu: 50m\n        memory: 64Mi\n", "  containers: []\n", 1), id: "vm-1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := RenderManifestWithID(tc.spec, tc.id); !errors.Is(err, ErrInvalid) {
				t.Fatalf("expected ErrInvalid, got %v", err)
			}
		})
	}
}
