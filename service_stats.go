package infra

// ServiceStats contains service statistics.
type ServiceStats struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	NumRequests  int    `json:"num_requests"`
	NumErrors    int    `json:"num_errors"`
	TotalLatency int64  `json:"total_latency_ms"`
	AvgLatency   int64  `json:"avg_latency_ms"`
}

// NodeInfo contains one online node's exposed service set.
type NodeInfo struct {
	Project  string   `json:"project"`
	Node     string   `json:"node"`
	Role     string   `json:"role"`
	Profile  string   `json:"profile"`
	Services []string `json:"services"`
	Updated  int64    `json:"updated"`
}

// ServiceNode indicates one node serving a service.
type ServiceNode struct {
	Node    string `json:"node"`
	Role    string `json:"role"`
	Profile string `json:"profile"`
}

// ServiceInfo is a service-centric online view.
type ServiceInfo struct {
	Service   string        `json:"service"`
	Name      string        `json:"name"`
	Desc      string        `json:"desc"`
	Updated   int64         `json:"updated"`
	Instances int           `json:"instances"`
	Nodes     []ServiceNode `json:"nodes"`
}
