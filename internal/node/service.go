package node

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"

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
}

type nodeService struct {
	sync.RWMutex
	nodes   map[string]*types.Node
	service micro.Service
}

func NewService() Service {
	// 创建带有mdns注册的服务
	service := micro.NewService(
		micro.Name("virtualization.service.node"),
		micro.Version("latest"),
		micro.Address(":9001"),
	)

	// 初始化服务
	service.Init()

	// 确保broker已启动
	if err := service.Options().Broker.Init(); err != nil {
		log.Fatalf("Broker init error: %v", err)
	}
	if err := service.Options().Broker.Connect(); err != nil {
		log.Fatalf("Broker connect error: %v", err)
	}

	ns := &nodeService{
		nodes:   make(map[string]*types.Node),
		service: service,
	}

	// 订阅节点发现事件
	if err := ns.subscribeNodeDiscovery(); err != nil {
		log.Fatalf("Failed to subscribe to node discovery: %v", err)
	}

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
