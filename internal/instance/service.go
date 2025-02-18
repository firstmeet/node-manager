package instance

import (
	"context"
	"errors"
	"sync"

	"virtualization-platform/internal/node"
	"virtualization-platform/internal/scheduler"
	"virtualization-platform/internal/types"

	"github.com/google/uuid"
	"go-micro.dev/v5"
	"go-micro.dev/v5/registry"
)

var (
	ErrInstanceNotFound = errors.New("instance not found")
	ErrInvalidType      = errors.New("invalid instance type")
)

type Service interface {
	CreateInstance(ctx context.Context, req *CreateInstanceRequest) (*types.Instance, error)
	DeleteInstance(ctx context.Context, id string) error
	GetInstance(ctx context.Context, id string) (*types.Instance, error)
	ListInstances(ctx context.Context) ([]*types.Instance, error)
}

type CreateInstanceRequest struct {
	Name     string
	Type     string // docker or libvirt
	Metadata map[string]string
}

type instanceService struct {
	sync.RWMutex
	instances map[string]*types.Instance
	scheduler scheduler.Service
	node      node.Service
	service   micro.Service
}

func NewService(scheduler scheduler.Service, nodeService node.Service) Service {
	service := micro.NewService(
		micro.Name("virtualization.service.instance"),
		micro.Version("latest"),
		micro.Registry(registry.DefaultRegistry),
		micro.Address(":9002"),
	)

	service.Init()

	return &instanceService{
		instances: make(map[string]*types.Instance),
		scheduler: scheduler,
		node:      nodeService,
		service:   service,
	}
}

func (s *instanceService) CreateInstance(ctx context.Context, req *CreateInstanceRequest) (*types.Instance, error) {
	if req.Type != "docker" && req.Type != "libvirt" {
		return nil, ErrInvalidType
	}

	// 使用调度器选择节点
	nodeID, err := s.scheduler.Schedule(ctx, &types.Instance{
		Type:     req.Type,
		Metadata: req.Metadata,
	})
	if err != nil {
		return nil, err
	}

	instance := &types.Instance{
		ID:       uuid.New().String(),
		Name:     req.Name,
		Type:     req.Type,
		Status:   "creating",
		NodeID:   nodeID,
		Metadata: req.Metadata,
	}

	s.Lock()
	s.instances[instance.ID] = instance
	s.Unlock()

	return instance, nil
}

func (s *instanceService) DeleteInstance(ctx context.Context, id string) error {
	s.Lock()
	defer s.Unlock()

	if _, exists := s.instances[id]; !exists {
		return ErrInstanceNotFound
	}

	delete(s.instances, id)
	return nil
}

func (s *instanceService) GetInstance(ctx context.Context, id string) (*types.Instance, error) {
	s.RLock()
	defer s.RUnlock()

	instance, exists := s.instances[id]
	if !exists {
		return nil, ErrInstanceNotFound
	}

	return instance, nil
}

func (s *instanceService) ListInstances(ctx context.Context) ([]*types.Instance, error) {
	s.RLock()
	defer s.RUnlock()

	instances := make([]*types.Instance, 0, len(s.instances))
	for _, instance := range s.instances {
		instances = append(instances, instance)
	}

	return instances, nil
}
