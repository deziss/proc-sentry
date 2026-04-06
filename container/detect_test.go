package container

import (
	"testing"
)

func TestDetectFromPath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		wantRuntime string
		wantID      string
	}{
		{
			name:        "host_process",
			path:        "/user.slice/user-1000.slice/session-1.scope",
			wantRuntime: "host",
			wantID:      "",
		},
		{
			name:        "docker_container",
			path:        "/docker/a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			wantRuntime: "docker",
			wantID:      "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
		},
		{
			name:        "kubernetes_pod",
			path:        "/kubepods/burstable/podXYZabc-defg/a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			wantRuntime: "kubernetes",
			wantID:      "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
		},
		{
			name:        "containerd",
			path:        "/system.slice/containerd.service/a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			wantRuntime: "containerd",
			wantID:      "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
		},
		{
			name:        "docker_short_id",
			path:        "/docker/a1b2c3d4e5f6",
			wantRuntime: "docker",
			wantID:      "a1b2c3d4e5f6",
		},
		{
			name:        "empty_path",
			path:        "",
			wantRuntime: "host",
			wantID:      "",
		},
		{
			name:        "no_container_markers",
			path:        "/sys/fs/cgroup/cpu",
			wantRuntime: "host",
			wantID:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := DetectFromPath(tt.path)
			if info.Runtime != tt.wantRuntime {
				t.Errorf("runtime: want %q, got %q", tt.wantRuntime, info.Runtime)
			}
			if info.ContainerID != tt.wantID {
				t.Errorf("containerID: want %q, got %q", tt.wantID, info.ContainerID)
			}
		})
	}
}
