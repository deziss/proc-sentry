package container

import (
	"regexp"
	"strings"

	"github.com/prometheus/procfs"
)

var reContainerID = regexp.MustCompile(`([0-9a-fA-F]{64}|[0-9a-fA-F]{12})`)

type Info struct {
	Runtime     string // "host", "docker", "kubernetes", "containerd"
	ContainerID string
}

// Detect reads cgroup info for a process and returns runtime/container details.
func Detect(p procfs.Proc) Info {
	info := Info{Runtime: "host"}

	cgroups, err := p.Cgroups()
	if err != nil {
		return info
	}

	for _, cg := range cgroups {
		path := cg.Path
		if !strings.Contains(path, "docker") && !strings.Contains(path, "containerd") && !strings.Contains(path, "kubepods") {
			continue
		}

		matches := reContainerID.FindStringSubmatch(path)
		if len(matches) > 1 {
			info.ContainerID = strings.ToLower(matches[1])
		}

		if strings.Contains(path, "docker") {
			info.Runtime = "docker"
		} else if strings.Contains(path, "kubepods") {
			info.Runtime = "kubernetes"
		} else if strings.Contains(path, "containerd") {
			info.Runtime = "containerd"
		}

		if info.ContainerID != "" {
			break
		}
	}

	return info
}

// DetectFromPath parses a cgroup path string directly (for testing).
func DetectFromPath(path string) Info {
	info := Info{Runtime: "host"}

	if !strings.Contains(path, "docker") && !strings.Contains(path, "containerd") && !strings.Contains(path, "kubepods") {
		return info
	}

	matches := reContainerID.FindStringSubmatch(path)
	if len(matches) > 1 {
		info.ContainerID = strings.ToLower(matches[1])
	}

	if strings.Contains(path, "docker") {
		info.Runtime = "docker"
	} else if strings.Contains(path, "kubepods") {
		info.Runtime = "kubernetes"
	} else if strings.Contains(path, "containerd") {
		info.Runtime = "containerd"
	}

	return info
}
