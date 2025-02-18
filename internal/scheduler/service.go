package scheduler

import (
	"context"
	"errors"

	"virtualization-platform/internal/node"
	"virtualization-platform/internal/types"
)

var (
	ErrNoAvailableNodes = errors.New("no available nodes")
)

type Service interface {
	Schedule(ctx context.Context, instance *types.Instance) (string, error)
}

type scheduler struct {
	nodeService node.Service
}

func NewService(nodeService node.Service) Service {
	return &scheduler{
		nodeService: nodeService,
	}
}

func (s *scheduler) Schedule(ctx context.Context, instance *types.Instance) (string, error) {
	nodes, err := s.nodeService.ListNodes(ctx)
	if err != nil {
		return "", err
	}

	if len(nodes) == 0 {
		return "", ErrNoAvailableNodes
	}

	// 选择负载最小的节点
	var selectedNode *types.Node
	var minLoad float64 = 1.0

	for _, node := range nodes {
		load := calculateNodeLoad(node)
		if load < minLoad {
			minLoad = load
			selectedNode = node
		}
	}

	return selectedNode.ID, nil
}

func calculateNodeLoad(node *types.Node) float64 {
	cpuLoad := float64(node.Resources.UsedCPU) / float64(node.Resources.TotalCPU)
	memLoad := float64(node.Resources.UsedMemory) / float64(node.Resources.TotalMemory)
	return (cpuLoad + memLoad) / 2
}
