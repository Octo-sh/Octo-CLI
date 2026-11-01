package intelligence

import (
	"fmt"
	"net"
	"sort"
)

// PortAssignment describes a host port selected for an application component.
type PortAssignment struct {
	Component string `json:"component" yaml:"component"`
	Requested int    `json:"requested" yaml:"requested"`
	Resolved  int    `json:"resolved" yaml:"resolved"`
	Automatic bool   `json:"automatic" yaml:"automatic"`
	Strict    bool   `json:"strict,omitempty" yaml:"strict,omitempty"`
}

// PortConflict represents a port collision on a host TCP port.
type PortConflict struct {
	Port        int  `json:"port" yaml:"port"`
	Strict      bool `json:"strict" yaml:"strict"`
	Alternative int  `json:"alternative,omitempty" yaml:"alternative,omitempty"`
}

// PortConflictError indicates a failure to allocate an expected TCP port.
type PortConflictError struct {
	Port     int
	Strict   bool
	NextFree int
}

func (e *PortConflictError) Error() string {
	if e.Strict {
		if e.NextFree > 0 {
			return fmt.Sprintf("requested TCP port %d is unavailable and the application declares it strictly (next free: %d)", e.Port, e.NextFree)
		}
		return fmt.Sprintf("requested TCP port %d is unavailable and the application declares it strictly", e.Port)
	}
	return fmt.Sprintf("no available TCP port found starting at %d", e.Port)
}

// FindNextFreePort searches for the next available TCP port starting from start.
func FindNextFreePort(start int, reserved map[int]struct{}) int {
	if start <= 0 {
		start = 3000
	}
	for port := start; port < 65536; port++ {
		if reserved != nil {
			if _, exists := reserved[port]; exists {
				continue
			}
		}
		if portAvailable(port) {
			return port
		}
	}
	return 0
}

// CheckPortConflict checks whether a requested port is occupied and finds an alternative if so.
func CheckPortConflict(requested int, reserved map[int]struct{}) (PortConflict, bool) {
	if requested <= 0 {
		return PortConflict{}, false
	}
	if reserved != nil {
		if _, exists := reserved[requested]; exists {
			alt := FindNextFreePort(requested+1, reserved)
			return PortConflict{Port: requested, Strict: false, Alternative: alt}, true
		}
	}
	if !portAvailable(requested) {
		alt := FindNextFreePort(requested+1, reserved)
		return PortConflict{Port: requested, Strict: false, Alternative: alt}, true
	}
	return PortConflict{Port: requested}, false
}

// PortAllocator finds deterministic free TCP ports.
type PortAllocator struct {
	StartOffset int
}

func (a PortAllocator) Allocate(requested int, reserved map[int]struct{}, strict bool) (int, bool, error) {
	if requested <= 0 {
		return 0, false, nil
	}
	if strict {
		if _, exists := reserved[requested]; exists || !portAvailable(requested) {
			alt := FindNextFreePort(requested+1, reserved)
			return 0, false, &PortConflictError{
				Port:     requested,
				Strict:   true,
				NextFree: alt,
			}
		}
		return requested, false, nil
	}
	start := requested
	if a.StartOffset > 0 {
		start += a.StartOffset
	}
	for port := start; port < 65536; port++ {
		if _, exists := reserved[port]; exists {
			continue
		}
		if portAvailable(port) {
			return port, port != requested, nil
		}
	}
	return 0, false, &PortConflictError{
		Port:     start,
		Strict:   false,
		NextFree: 0,
	}
}

func portAvailable(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = listener.Close()
	return true
}

func AllocateComponentPorts(components []Component) ([]PortAssignment, error) {
	return AllocateComponentPortsWithServices(components, nil)
}

// AllocateComponentPortsWithServices reserves service ports before allocating component ports,
// coordinating dynamic port shifting when conflicts occur.
func AllocateComponentPortsWithServices(components []Component, services []Service) ([]PortAssignment, error) {
	ordered := append([]Component(nil), components...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Name < ordered[j].Name })

	reserved := make(map[int]struct{})
	for _, svc := range services {
		for _, p := range svc.Ports {
			if port := extractHostPort(p); port > 0 {
				reserved[port] = struct{}{}
			}
		}
	}

	assignments := make([]PortAssignment, 0)
	allocator := PortAllocator{}

	for _, component := range ordered {
		if component.Port <= 0 {
			continue
		}
		port, automatic, err := allocator.Allocate(component.Port, reserved, component.PortStrict)
		if err != nil {
			return nil, fmt.Errorf("allocate port for component %q: %w", component.Name, err)
		}
		reserved[port] = struct{}{}
		assignments = append(assignments, PortAssignment{
			Component: component.Name,
			Requested: component.Port,
			Resolved:  port,
			Automatic: automatic,
			Strict:    component.PortStrict,
		})
	}
	return assignments, nil
}
