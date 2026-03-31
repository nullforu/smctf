package stack

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	stackv1 "smctf/internal/gen/stack/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GRPCClient struct {
	addr    string
	apiKey  string
	timeout time.Duration
	conn    *grpc.ClientConn
	client  stackv1.StackServiceClient
}

func NewGRPCClient(addr, apiKey string, timeout time.Duration) (*GRPCClient, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil, fmt.Errorf("grpc addr is required")
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc dial: %w", err)
	}

	return &GRPCClient{
		addr:    addr,
		apiKey:  apiKey,
		timeout: timeout,
		conn:    conn,
		client:  stackv1.NewStackServiceClient(conn),
	}, nil
}

func (c *GRPCClient) Close() error {
	if c.conn == nil {
		return nil
	}

	return c.conn.Close()
}

func (c *GRPCClient) CreateStack(ctx context.Context, targetPorts []TargetPortSpec, podSpec string) (*StackInfo, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	ctx = c.withAPIKey(ctx)
	resp, err := c.client.CreateStack(ctx, &stackv1.CreateStackRequest{
		PodSpec:     podSpec,
		TargetPorts: toProtoTargetPorts(targetPorts),
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}

	stack := resp.GetStack()
	if stack == nil {
		return nil, ErrUnexpected
	}

	return toStackInfo(stack), nil
}

func (c *GRPCClient) GetStackStatus(ctx context.Context, stackID string) (*StackStatus, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	ctx = c.withAPIKey(ctx)
	resp, err := c.client.GetStackStatusSummary(ctx, &stackv1.GetStackStatusSummaryRequest{StackId: stackID})
	if err != nil {
		return nil, mapGRPCError(err)
	}

	summary := resp.GetSummary()
	if summary == nil {
		return nil, ErrUnexpected
	}

	return toStackStatus(summary), nil
}

func (c *GRPCClient) DeleteStack(ctx context.Context, stackID string) error {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	ctx = c.withAPIKey(ctx)
	_, err := c.client.DeleteStack(ctx, &stackv1.DeleteStackRequest{StackId: stackID})
	if err != nil {
		return mapGRPCError(err)
	}

	return nil
}

func (c *GRPCClient) withAPIKey(ctx context.Context) context.Context {
	if strings.TrimSpace(c.apiKey) == "" {
		return ctx
	}

	return metadata.AppendToOutgoingContext(ctx, "x-api-key", c.apiKey)
}

func (c *GRPCClient) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.timeout <= 0 {
		return ctx, func() {}
	}

	if deadline, ok := ctx.Deadline(); ok {
		desired := time.Now().Add(c.timeout)
		if deadline.Before(desired) {
			return ctx, func() {}
		}
		return context.WithDeadline(ctx, desired)
	}

	return context.WithTimeout(ctx, c.timeout)
}

func mapGRPCError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrUnavailable
	}

	if errors.Is(err, context.Canceled) {
		return ErrUnexpected
	}

	st, ok := status.FromError(err)
	if !ok {
		return ErrUnexpected
	}

	switch st.Code() {
	case codes.NotFound:
		return ErrNotFound
	case codes.InvalidArgument:
		return ErrInvalid
	case codes.Unavailable, codes.DeadlineExceeded:
		return ErrUnavailable
	default:
		return ErrUnexpected
	}
}

func toProtoTargetPorts(ports []TargetPortSpec) []*stackv1.PortSpec {
	out := make([]*stackv1.PortSpec, 0, len(ports))
	for _, port := range ports {
		out = append(out, &stackv1.PortSpec{
			ContainerPort: int32(port.ContainerPort),
			Protocol:      port.Protocol,
		})
	}

	return out
}

func toStackInfo(stack *stackv1.Stack) *StackInfo {
	ports := make([]PortMapping, 0, len(stack.Ports))
	for _, port := range stack.Ports {
		ports = append(ports, PortMapping{
			ContainerPort: int(port.ContainerPort),
			Protocol:      port.Protocol,
			NodePort:      int(port.NodePort),
		})
	}

	return &StackInfo{
		StackID:              stack.StackId,
		PodID:                stack.PodId,
		Namespace:            stack.Namespace,
		NodeID:               stack.NodeId,
		NodePublicIP:         stack.GetNodePublicIp(),
		PodSpec:              stack.PodSpec,
		Ports:                ports,
		ServiceName:          stack.ServiceName,
		Status:               statusToString(stack.Status),
		TTLExpiresAt:         timeOrZero(stack.TtlExpiresAt),
		CreatedAt:            timeOrZero(stack.CreatedAt),
		UpdatedAt:            timeOrZero(stack.UpdatedAt),
		RequestedCPUMilli:    int(stack.RequestedCpuMilli),
		RequestedMemoryBytes: int(stack.RequestedMemoryBytes),
	}
}

func toStackStatus(summary *stackv1.StackStatusSummary) *StackStatus {
	ports := make([]PortMapping, 0, len(summary.Ports))
	for _, port := range summary.Ports {
		ports = append(ports, PortMapping{
			ContainerPort: int(port.ContainerPort),
			Protocol:      port.Protocol,
			NodePort:      int(port.NodePort),
		})
	}

	return &StackStatus{
		StackID:      summary.StackId,
		Status:       statusToString(summary.Status),
		TTL:          timeOrZero(summary.Ttl),
		Ports:        ports,
		NodePublicIP: summary.GetNodePublicIp(),
	}
}

func timeOrZero(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}

	return ts.AsTime()
}

func statusToString(status stackv1.Status) string {
	switch status {
	case stackv1.Status_STATUS_CREATING:
		return "creating"
	case stackv1.Status_STATUS_RUNNING:
		return "running"
	case stackv1.Status_STATUS_STOPPED:
		return "stopped"
	case stackv1.Status_STATUS_FAILED:
		return "failed"
	case stackv1.Status_STATUS_NODE_DELETED:
		return "node_deleted"
	default:
		return ""
	}
}
