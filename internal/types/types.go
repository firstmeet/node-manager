package types

type Instance struct {
	ID       string
	Name     string
	Type     string // docker or libvirt
	Status   string
	NodeID   string
	Metadata map[string]string
}

type Node struct {
	ID        string
	Hostname  string
	IP        string
	Resources Resources
	Status    string
}

type Resources struct {
	TotalCPU     int64
	UsedCPU      int64
	TotalMemory  int64
	UsedMemory   int64
	TotalStorage int64
	UsedStorage  int64
} 