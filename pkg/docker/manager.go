package docker

import (
	"context"
	"errors"

	"virtualization-platform/internal/types"

	typesDocker "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

var (
	ErrContainerNotFound = errors.New("container not found")
	ErrInvalidConfig     = errors.New("invalid container configuration")
)

type Manager interface {
	CreateContainer(ctx context.Context, instance *types.Instance) error
	DeleteContainer(ctx context.Context, instanceID string) error
	GetContainerStatus(ctx context.Context, instanceID string) (string, error)
	ListContainers(ctx context.Context) ([]*types.Instance, error)
}

type dockerManager struct {
	client *client.Client
}

func NewManager() (Manager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	return &dockerManager{
		client: cli,
	}, nil
}

func (m *dockerManager) CreateContainer(ctx context.Context, instance *types.Instance) error {
	// 从实例元数据中获取容器配置
	image := instance.Metadata["image"]
	if image == "" {
		return ErrInvalidConfig
	}

	// 创建容器配置
	config := &container.Config{
		Image: image,
		Labels: map[string]string{
			"instance_id": instance.ID,
		},
	}

	// 创建容器
	resp, err := m.client.ContainerCreate(ctx, config, nil, nil, nil, instance.ID)
	if err != nil {
		return err
	}

	// 启动容器
	return m.client.ContainerStart(ctx, resp.ID, typesDocker.ContainerStartOptions{})
}

func (m *dockerManager) DeleteContainer(ctx context.Context, instanceID string) error {
	containers, err := m.client.ContainerList(ctx, typesDocker.ContainerListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", "instance_id="+instanceID)),
	})
	if err != nil {
		return err
	}

	if len(containers) == 0 {
		return ErrContainerNotFound
	}

	return m.client.ContainerRemove(ctx, containers[0].ID, typesDocker.ContainerRemoveOptions{
		Force: true,
	})
}

func (m *dockerManager) GetContainerStatus(ctx context.Context, instanceID string) (string, error) {
	containers, err := m.client.ContainerList(ctx, typesDocker.ContainerListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", "instance_id="+instanceID)),
	})
	if err != nil {
		return "", err
	}

	if len(containers) == 0 {
		return "", ErrContainerNotFound
	}

	return containers[0].State, nil
}

func (m *dockerManager) ListContainers(ctx context.Context) ([]*types.Instance, error) {
	containers, err := m.client.ContainerList(ctx, typesDocker.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	instances := make([]*types.Instance, 0, len(containers))
	for _, container := range containers {
		if instanceID, ok := container.Labels["instance_id"]; ok {
			instances = append(instances, &types.Instance{
				ID:     instanceID,
				Type:   "docker",
				Status: container.State,
			})
		}
	}

	return instances, nil
}
