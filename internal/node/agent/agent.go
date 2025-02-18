package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"virtualization-platform/internal/node"
	"virtualization-platform/internal/types"

	"github.com/google/uuid"
	"go-micro.dev/v5"
	"go-micro.dev/v5/broker"
)

type Agent struct {
	node     *types.Node
	service  micro.Service
	hostname string
}

func NewAgent() (*Agent, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	service := micro.NewService(
		micro.Name("virtualization.agent.node."+hostname),
		micro.Version("latest"),
		micro.Address(":0"),
	)

	service.Init()

	// 确保broker已启动
	if err := service.Options().Broker.Init(); err != nil {
		return nil, fmt.Errorf("broker init error: %v", err)
	}
	if err := service.Options().Broker.Connect(); err != nil {
		return nil, fmt.Errorf("broker connect error: %v", err)
	}

	return &Agent{
		service:  service,
		hostname: hostname,
		node: &types.Node{
			ID:       uuid.New().String(),
			Hostname: hostname,
			Status:   "ready",
			Resources: types.Resources{
				TotalCPU:     8, // 这里可以根据实际系统获取
				TotalMemory:  16384,
				TotalStorage: 1024000,
			},
		},
	}, nil
}

func (a *Agent) Start() error {
	// 定期发布节点信息
	go a.publishNodeInfo()

	return a.service.Run()
}

func (a *Agent) publishNodeInfo() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			event := &node.NodeDiscoveryEvent{
				Node: a.node,
			}

			data, err := json.Marshal(event)
			if err != nil {
				continue
			}

			// 使用Broker发布消息
			if err := a.service.Options().Broker.Publish(node.NodeDiscoveryTopic, &broker.Message{
				Body: data,
			}); err != nil {
				continue
			}
		}
	}
}
