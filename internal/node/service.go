package node

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"sync"
	"time"

	"virtualization-platform/internal/monitor"
	"virtualization-platform/internal/types"

	"github.com/google/uuid"
	"go-micro.dev/v5"
	"go-micro.dev/v5/broker"
)

const (
	NodeDiscoveryTopic = "node.discovery"
)

// NodeDiscoveryEvent 节点发现事件
type NodeDiscoveryEvent struct {
	Node *types.Node `json:"node"`
}

var (
	ErrNodeNotFound = errors.New("node not found")
)

type Service interface {
	RegisterNode(ctx context.Context, node *types.Node) error
	GetNode(ctx context.Context, id string) (*types.Node, error)
	ListNodes(ctx context.Context) ([]*types.Node, error)
	PublishNodeInfo(ctx context.Context) error
	StartLocalAgent() error
}

type nodeService struct {
	sync.RWMutex
	nodes     map[string]*types.Node
	service   micro.Service
	localNode *types.Node
	monitor   monitor.ResourceMonitor
}

func NewService() Service {
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Failed to get hostname: %v", err)
	}

	service := micro.NewService(
		micro.Name("virtualization.service.node"),
		micro.Version("latest"),
		micro.Address(":9001"),
	)

	service.Init()

	if err := service.Options().Broker.Init(); err != nil {
		log.Fatalf("Broker init error: %v", err)
	}
	if err := service.Options().Broker.Connect(); err != nil {
		log.Fatalf("Broker connect error: %v", err)
	}

	// 创建资源监控器
	resourceMonitor := monitor.NewMonitor("/")

	// 创建本地节点信息
	localNode := &types.Node{
		ID:       uuid.New().String(),
		Hostname: hostname,
		Status:   "ready",
		Resources: types.Resources{
			TotalCPU:     8,
			TotalMemory:  16384,
			TotalStorage: 1024000,
		},
	}

	ns := &nodeService{
		nodes:     make(map[string]*types.Node),
		service:   service,
		localNode: localNode,
		monitor:   resourceMonitor,
	}

	// 订阅节点发现事件
	if err := ns.subscribeNodeDiscovery(); err != nil {
		log.Fatalf("Failed to subscribe to node discovery: %v", err)
	}

	// 注册本地节点
	if err := ns.RegisterNode(context.Background(), localNode); err != nil {
		log.Fatalf("Failed to register local node: %v", err)
	}

	// 启动资源监控和节点发布
	go ns.StartLocalAgent()

	return ns
}

// 订阅节点发现事件
func (s *nodeService) subscribeNodeDiscovery() error {
	_, err := s.service.Options().Broker.Subscribe(NodeDiscoveryTopic, func(e broker.Event) error {
		var event NodeDiscoveryEvent
		if err := json.Unmarshal(e.Message().Body, &event); err != nil {
			return err
		}

		if event.Node != nil {
			return s.RegisterNode(context.Background(), event.Node)
		}
		return nil
	})

	return err
}

// 发布节点信息
func (s *nodeService) PublishNodeInfo(ctx context.Context) error {
	s.RLock()
	defer s.RUnlock()

	for _, node := range s.nodes {
		event := &NodeDiscoveryEvent{
			Node: node,
		}

		data, err := json.Marshal(event)
		if err != nil {
			return err
		}

		if err := s.service.Options().Broker.Publish(NodeDiscoveryTopic, &broker.Message{
			Body: data,
		}); err != nil {
			return err
		}
	}

	return nil
}

// 实现 Service 接口
func (s *nodeService) RegisterNode(ctx context.Context, node *types.Node) error {
	s.Lock()
	defer s.Unlock()

	// 如果节点ID为空，生成新的UUID
	if node.ID == "" {
		node.ID = uuid.New().String()
	}
	log.Printf("RegisterNode: %v", node)

	s.nodes[node.ID] = node
	return nil
}

func (s *nodeService) GetNode(ctx context.Context, id string) (*types.Node, error) {
	s.RLock()
	defer s.RUnlock()

	node, exists := s.nodes[id]
	if !exists {
		return nil, ErrNodeNotFound
	}
	return node, nil
}

func (s *nodeService) ListNodes(ctx context.Context) ([]*types.Node, error) {
	s.RLock()
	defer s.RUnlock()

	nodes := make([]*types.Node, 0, len(s.nodes))
	for _, node := range s.nodes {
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// StartLocalAgent 启动本地节点代理
func (s *nodeService) StartLocalAgent() error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 更新资源使用情况
			cpuUsage, err := s.monitor.GetCPUUsage()
			if err != nil {
				log.Printf("Failed to get CPU usage: %v", err)
				continue
			}

			memUsage, err := s.monitor.GetMemoryUsage()
			if err != nil {
				log.Printf("Failed to get memory usage: %v", err)
				continue
			}

			storageUsage, err := s.monitor.GetStorageUsage()
			if err != nil {
				log.Printf("Failed to get storage usage: %v", err)
				continue
			}

			s.Lock()
			s.localNode.Resources.UsedCPU = int64(cpuUsage)
			s.localNode.Resources.UsedMemory = int64(memUsage)
			s.localNode.Resources.UsedStorage = int64(storageUsage)
			s.Unlock()

			// 发布更新后的节点信息
			event := &NodeDiscoveryEvent{
				Node: s.localNode,
			}

			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("Failed to marshal node info: %v", err)
				continue
			}

			if err := s.service.Options().Broker.Publish(NodeDiscoveryTopic, &broker.Message{
				Body: data,
			}); err != nil {
				log.Printf("Failed to publish node info: %v", err)
				continue
			}

			log.Printf("Published local node info: CPU: %.2f%%, Memory: %d MB, Storage: %d MB",
				cpuUsage, memUsage/1024/1024, storageUsage/1024/1024)
		}
	}
}
