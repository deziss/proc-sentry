package collector

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/prometheus/procfs"
)

const tcpListen = 10 // 0x0A = TCP_LISTEN state

// ResolvePortsSafely reads listening TCP ports from a process's network namespace.
func ResolvePortsSafely(p *procfs.Proc, procFSPath string) string {
	fds, err := p.FileDescriptorTargets()
	if err != nil {
		return ""
	}

	validInodes := make(map[uint64]bool)
	for _, fd := range fds {
		if strings.HasPrefix(fd, "socket:[") {
			inodeStr := strings.TrimSuffix(strings.TrimPrefix(fd, "socket:["), "]")
			if inode, err := strconv.ParseUint(inodeStr, 10, 64); err == nil {
				validInodes[inode] = true
			}
		}
	}
	if len(validInodes) == 0 {
		return ""
	}

	netFS, err := procfs.NewFS(filepath.Join(procFSPath, strconv.Itoa(p.PID)))
	if err != nil {
		return ""
	}

	var netSockList []uint64

	tcp, err := netFS.NetTCP()
	if err == nil && tcp != nil {
		for _, stat := range tcp {
			if stat.St == tcpListen && validInodes[stat.Inode] {
				netSockList = append(netSockList, stat.LocalPort)
			}
		}
	}

	tcp6, err := netFS.NetTCP6()
	if err == nil && tcp6 != nil {
		for _, stat := range tcp6 {
			if stat.St == tcpListen && validInodes[stat.Inode] {
				netSockList = append(netSockList, stat.LocalPort)
			}
		}
	}

	if len(netSockList) == 0 {
		return ""
	}

	seen := make(map[uint64]bool)
	var ports []int
	for _, p := range netSockList {
		if !seen[p] {
			seen[p] = true
			ports = append(ports, int(p))
		}
	}
	sort.Ints(ports)

	var parts []string
	for _, p := range ports {
		parts = append(parts, strconv.Itoa(p))
	}
	return strings.Join(parts, ",")
}
