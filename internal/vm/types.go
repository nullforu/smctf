package vm

import "time"

type PortSpec struct {
	HostPort      int    `json:"host_port,omitempty" yaml:"host_port,omitempty"`
	ContainerPort int    `json:"container_port" yaml:"container_port"`
	Protocol      string `json:"protocol,omitempty" yaml:"protocol,omitempty"`
}

type PortMapping struct {
	HostPort      int    `json:"host_port"`
	NodePort      int    `json:"node_port,omitempty"`
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol"`
}

type PortMappings []PortMapping

type ResourceSpec struct {
	CPU    string `json:"cpu" yaml:"cpu"`
	Memory string `json:"memory" yaml:"memory"`
}

type ContainerSpec struct {
	Name     string       `json:"name" yaml:"name"`
	Image    string       `json:"image" yaml:"image"`
	Args     []string     `json:"args,omitempty" yaml:"args,omitempty"`
	Env      []string     `json:"env,omitempty" yaml:"env,omitempty"`
	WorkDir  string       `json:"work_dir,omitempty" yaml:"work_dir,omitempty"`
	Resource ResourceSpec `json:"resource" yaml:"resource"`
}

type SandboxSpec struct {
	Egress     bool            `json:"egress" yaml:"egress"`
	TTLSeconds int64           `json:"ttl_seconds,omitempty" yaml:"ttl_seconds,omitempty"`
	Ports      []PortSpec      `json:"ports,omitempty" yaml:"ports,omitempty"`
	Containers []ContainerSpec `json:"containers" yaml:"containers"`
}

type CreateSandboxRequest struct {
	ID   string      `json:"id" yaml:"id"`
	Spec SandboxSpec `json:"spec" yaml:"spec"`
}

type SandboxStatus struct {
	Phase         string        `json:"phase"`
	NodeName      string        `json:"node_name,omitempty"`
	ExternalIP    string        `json:"external,omitempty"`
	AssignedPorts []PortMapping `json:"assigned_ports,omitempty"`
	ExpireAt      *time.Time    `json:"expire_at,omitempty"`
	LastError     string        `json:"last_error,omitempty"`
}

type Sandbox struct {
	ID        string        `json:"id"`
	Spec      SandboxSpec   `json:"spec"`
	Status    SandboxStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type sandboxObjectResponse struct {
	Sandbox *Sandbox `json:"sandbox"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
