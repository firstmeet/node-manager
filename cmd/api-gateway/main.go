package main

import (
	"log"

	"virtualization-platform/internal/instance"
	"virtualization-platform/internal/node"
	"virtualization-platform/internal/scheduler"
	"virtualization-platform/internal/types"

	"github.com/gin-gonic/gin"
	"go-micro.dev/v5"
	"go-micro.dev/v5/registry"
)

func main() {
	// 创建micro服务
	service := micro.NewService(
		micro.Name("virtualization.service.gateway"),
		micro.Version("latest"),
		// 使用mdns注册
		micro.Registry(registry.DefaultRegistry),
	)

	// 初始化服务
	service.Init()

	// 创建服务客户端
	nodeService := node.NewService()
	schedulerService := scheduler.NewService(nodeService)
	instanceService := instance.NewService(schedulerService, nodeService)

	// 创建Gin路由
	r := gin.Default()

	// 实例相关API
	instances := r.Group("/api/v1/instances")
	{
		instances.POST("/", createInstance(instanceService))
		instances.GET("/", listInstances(instanceService))
		instances.GET("/:id", getInstance(instanceService))
		instances.DELETE("/:id", deleteInstance(instanceService))
	}

	// 节点相关API
	nodes := r.Group("/api/v1/nodes")
	{
		nodes.POST("/", registerNode(nodeService))
		nodes.GET("/", listNodes(nodeService))
		nodes.GET("/:id", getNode(nodeService))
	}

	// 启动HTTP服务器
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

// API处理函数
func createInstance(svc instance.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req instance.CreateInstanceRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		inst, err := svc.CreateInstance(c.Request.Context(), &req)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(201, inst)
	}
}

func listInstances(svc instance.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		instances, err := svc.ListInstances(c.Request.Context())
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, instances)
	}
}

func getInstance(svc instance.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		instance, err := svc.GetInstance(c.Request.Context(), id)
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, instance)
	}
}

func deleteInstance(svc instance.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := svc.DeleteInstance(c.Request.Context(), id); err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}

		c.Status(204)
	}
}

func listNodes(svc node.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodes, err := svc.ListNodes(c.Request.Context())
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, nodes)
	}
}

func getNode(svc node.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		node, err := svc.GetNode(c.Request.Context(), id)
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, node)
	}
}

// 添加节点注册处理函数
func registerNode(svc node.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var node types.Node
		if err := c.ShouldBindJSON(&node); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		if err := svc.RegisterNode(c.Request.Context(), &node); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(201, node)
	}
}
