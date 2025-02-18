package main

import (
	"log"

	"virtualization-platform/internal/node/agent"
)

func main() {
	agent, err := agent.NewAgent()
	if err != nil {
		log.Fatalf("Failed to create node agent: %v", err)
	}

	if err := agent.Start(); err != nil {
		log.Fatalf("Failed to start node agent: %v", err)
	}
} 