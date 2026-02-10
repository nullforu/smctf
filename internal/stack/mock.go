package stack

import (
	"context"
)

type MockClient struct {
	CreateStackFn    func(ctx context.Context, targetPort int, podSpec string) (*StackInfo, error)
	GetStackStatusFn func(ctx context.Context, stackID string) (*StackStatus, error)
	DeleteStackFn    func(ctx context.Context, stackID string) error
}

func (m *MockClient) CreateStack(ctx context.Context, targetPort int, podSpec string) (*StackInfo, error) {
	if m.CreateStackFn == nil {
		return nil, ErrUnexpected
	}

	return m.CreateStackFn(ctx, targetPort, podSpec)
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
