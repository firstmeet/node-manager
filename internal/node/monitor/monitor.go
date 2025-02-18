package monitor

import (
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/disk"
)

type ResourceMonitor interface {
	GetCPUUsage() (float64, error)
	GetMemoryUsage() (uint64, error)
	GetStorageUsage() (uint64, error)
}

type monitor struct {
	diskPath string
}

func NewMonitor(diskPath string) ResourceMonitor {
	return &monitor{
		diskPath: diskPath,
	}
}

func (m *monitor) GetCPUUsage() (float64, error) {
	percentage, err := cpu.Percent(0, false)
	if err != nil {
		return 0, err
	}
	return percentage[0], nil
}

func (m *monitor) GetMemoryUsage() (uint64, error) {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}
	return memInfo.Used, nil
}

func (m *monitor) GetStorageUsage() (uint64, error) {
	usage, err := disk.Usage(m.diskPath)
	if err != nil {
		return 0, err
	}
	return usage.Used, nil
} 