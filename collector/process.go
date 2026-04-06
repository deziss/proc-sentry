package collector

import (
	"os"
	"strconv"

	"github.com/deziss/proc-sentry/container"
	"github.com/deziss/proc-sentry/internal"
	"github.com/prometheus/procfs"
)

var pageSize = float64(os.Getpagesize())

// Process holds metrics for a single process.
type Process struct {
	PID         int
	User        string
	Command     string
	Runtime     string
	ContainerID string
	CPUPct      float64
	MemRSS      float64 // bytes
	DiskRead    float64
	DiskWrite   float64
	Ticks       float64
	Ports       string
	FullPath    string
	ProcfsProc  *procfs.Proc
}

// ReadProcess extracts metrics from a procfs process entry.
func ReadProcess(p procfs.Proc, userMap map[string]string, enableDiskIO bool) (*Process, error) {
	stat, err := p.Stat()
	if err != nil {
		return nil, err
	}

	status, err := p.NewStatus()
	if err != nil {
		return nil, err
	}

	uid := "0"
	if len(status.UIDs) > 0 {
		uid = strconv.FormatUint(status.UIDs[0], 10)
	}
	username := internal.ResolveUser(userMap, uid)

	var rd, wr float64
	if enableDiskIO {
		io, err := p.IO()
		if err == nil {
			rd = float64(io.ReadBytes)
			wr = float64(io.WriteBytes)
		}
	}

	rssBytes := float64(stat.RSS) * pageSize

	cmd, err := p.Comm()
	if err != nil {
		cmd = "unknown"
	}

	fullPath, err := p.Executable()
	if err != nil || fullPath == "" {
		fullPath = cmd
	}

	ci := container.Detect(p)

	return &Process{
		PID:         p.PID,
		User:        username,
		Command:     cmd,
		FullPath:    fullPath,
		Runtime:     ci.Runtime,
		ContainerID: ci.ContainerID,
		Ticks:       stat.CPUTime(),
		MemRSS:      rssBytes,
		DiskRead:    rd,
		DiskWrite:   wr,
		ProcfsProc:  &p,
	}, nil
}
