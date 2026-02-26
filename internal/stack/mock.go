package stack

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type MockClient struct {
	CreateStackFn    func(ctx context.Context, targetPorts []TargetPortSpec, podSpec string) (*StackInfo, error)
	GetStackStatusFn func(ctx context.Context, stackID string) (*StackStatus, error)
	DeleteStackFn    func(ctx context.Context, stackID string) error
}

func (m *MockClient) CreateStack(ctx context.Context, targetPorts []TargetPortSpec, podSpec string) (*StackInfo, error) {
	if m.CreateStackFn == nil {
		return nil, ErrUnexpected
	}

	return m.CreateStackFn(ctx, targetPorts, podSpec)
}

func (m *MockClient) GetStackStatus(ctx context.Context, stackID string) (*StackStatus, error) {
	if m.GetStackStatusFn == nil {
		return nil, ErrUnexpected
	}

	return m.GetStackStatusFn(ctx, stackID)
}

func (m *MockClient) DeleteStack(ctx context.Context, stackID string) error {
	if m.DeleteStackFn == nil {
		return ErrUnexpected
	}

	return m.DeleteStackFn(ctx, stackID)
}

type ProvisionerMock struct {
	mu     sync.Mutex
	nextID int
	stacks map[string]StackInfo

	createErr     error
	statusByID    map[string]string
	statusErrByID map[string]error
	deleteErrByID map[string]error
	deleteCalls   map[string]int
	createCalls   int
}

func NewProvisionerMock() *ProvisionerMock {
	return &ProvisionerMock{
		nextID:        1,
		stacks:        make(map[string]StackInfo),
		statusByID:    make(map[string]string),
		statusErrByID: make(map[string]error),
		deleteErrByID: make(map[string]error),
		deleteCalls:   make(map[string]int),
	}
}

func (p *ProvisionerMock) Client() *MockClient {
	return &MockClient{
		CreateStackFn: func(ctx context.Context, targetPorts []TargetPortSpec, podSpec string) (*StackInfo, error) {
			p.mu.Lock()
			if p.createErr != nil {
				err := p.createErr
				p.mu.Unlock()
				return nil, err
			}
			id := p.nextID
			p.nextID++
			p.createCalls++
			stackID := fmt.Sprintf("stack-test-%d", id)
			now := time.Now().UTC()
			portMappings := make([]PortMapping, 0, len(targetPorts))
			for idx, port := range targetPorts {
				portMappings = append(portMappings, PortMapping{
					ContainerPort: port.ContainerPort,
					Protocol:      port.Protocol,
					NodePort:      31001 + id + idx,
				})
			}
			info := StackInfo{
				StackID:      stackID,
				PodID:        stackID,
				Namespace:    "stacks",
				NodeID:       "dev-worker",
				NodePublicIP: "127.0.0.1",
				PodSpec:      podSpec,
				Ports:        portMappings,
				ServiceName:  "svc-" + stackID,
				Status:       "running",
				TTLExpiresAt: now.Add(2 * time.Hour),
				CreatedAt:    now,
				UpdatedAt:    now,
			}

			p.stacks[stackID] = info
			p.mu.Unlock()

			return &info, nil
		},
		GetStackStatusFn: func(ctx context.Context, stackID string) (*StackStatus, error) {
			p.mu.Lock()
			if err, ok := p.statusErrByID[stackID]; ok && err != nil {
				p.mu.Unlock()
				return nil, err
			}
			info, ok := p.stacks[stackID]
			status := info.Status
			if override, ok := p.statusByID[stackID]; ok && override != "" {
				status = override
			}
			p.mu.Unlock()
			if !ok {
				return nil, ErrNotFound
			}

			return &StackStatus{
				StackID:      info.StackID,
				Status:       status,
				TTL:          info.TTLExpiresAt,
				Ports:        info.Ports,
				NodePublicIP: info.NodePublicIP,
			}, nil
		},
		DeleteStackFn: func(ctx context.Context, stackID string) error {
			p.mu.Lock()
			if err, ok := p.deleteErrByID[stackID]; ok && err != nil {
				p.deleteCalls[stackID]++
				p.mu.Unlock()
				return err
			}
			_, ok := p.stacks[stackID]
			delete(p.stacks, stackID)
			p.deleteCalls[stackID]++
			p.mu.Unlock()

			if !ok {
				return ErrNotFound
			}

			return nil
		},
	}
}

func (p *ProvisionerMock) AddStack(info StackInfo) {
	p.mu.Lock()
	p.stacks[info.StackID] = info
	p.mu.Unlock()
}

func (p *ProvisionerMock) SetStatus(stackID, status string) {
	p.mu.Lock()
	p.statusByID[stackID] = status
	p.mu.Unlock()
}

func (p *ProvisionerMock) SetStatusError(stackID string, err error) {
	p.mu.Lock()
	p.statusErrByID[stackID] = err
	p.mu.Unlock()
}

func (p *ProvisionerMock) SetDeleteError(stackID string, err error) {
	p.mu.Lock()
	p.deleteErrByID[stackID] = err
	p.mu.Unlock()
}

func (p *ProvisionerMock) SetCreateError(err error) {
	p.mu.Lock()
	p.createErr = err
	p.mu.Unlock()
}

func (p *ProvisionerMock) DeleteCount(stackID string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.deleteCalls[stackID]
}

func (p *ProvisionerMock) CreateCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.createCalls
}
