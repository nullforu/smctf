package vm

import (
	"context"
	"sync"
	"time"
)

type MockClient struct {
	CreateSandboxFn func(ctx context.Context, id string, specYAML string) (*Sandbox, error)
	GetSandboxFn    func(ctx context.Context, id string) (*Sandbox, error)
	DeleteSandboxFn func(ctx context.Context, id string) error
}

type OrchestratorMock struct {
	mu        sync.Mutex
	sandboxes map[string]*Sandbox
}

func NewOrchestratorMock() *OrchestratorMock {
	return &OrchestratorMock{sandboxes: make(map[string]*Sandbox)}
}

func (m *OrchestratorMock) Client() API {
	return &MockClient{
		CreateSandboxFn: m.createSandbox,
		GetSandboxFn:    m.getSandbox,
		DeleteSandboxFn: m.deleteSandbox,
	}
}

func (m *OrchestratorMock) createSandbox(_ context.Context, id string, _ string) (*Sandbox, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	exp := time.Now().UTC().Add(time.Hour)
	s := &Sandbox{
		ID: id,
		Status: SandboxStatus{
			Phase:         "Running",
			ExternalIP:    "127.0.0.1",
			AssignedPorts: []PortMapping{{HostPort: 31001, ContainerPort: 80, Protocol: "TCP"}},
			ExpireAt:      &exp,
		},
	}
	m.sandboxes[id] = s
	return s, nil
}

func (m *OrchestratorMock) getSandbox(_ context.Context, id string) (*Sandbox, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sandboxes[id]
	if !ok {
		return nil, ErrNotFound
	}
	return s, nil
}

func (m *OrchestratorMock) deleteSandbox(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sandboxes, id)
	return nil
}

func (m *MockClient) CreateSandbox(ctx context.Context, id string, specYAML string) (*Sandbox, error) {
	if m.CreateSandboxFn != nil {
		return m.CreateSandboxFn(ctx, id, specYAML)
	}
	return nil, ErrUnexpected
}

func (m *MockClient) GetSandbox(ctx context.Context, id string) (*Sandbox, error) {
	if m.GetSandboxFn != nil {
		return m.GetSandboxFn(ctx, id)
	}
	return nil, ErrUnexpected
}

func (m *MockClient) DeleteSandbox(ctx context.Context, id string) error {
	if m.DeleteSandboxFn != nil {
		return m.DeleteSandboxFn(ctx, id)
	}
	return ErrUnexpected
}
