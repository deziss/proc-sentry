package collector

import "sort"

// SelectTopN returns the union of top-N processes across CPU, Memory, DiskRead, DiskWrite.
// At most 4*topN unique processes are returned.
func SelectTopN(list []*Process, topN int) map[int]*Process {
	winners := make(map[int]*Process)

	addTop := func(less func(i, j int) bool, val func(*Process) float64) {
		sort.Slice(list, less)
		for i := 0; i < topN && i < len(list); i++ {
			p := list[i]
			if val(p) > 0 {
				winners[p.PID] = p
			}
		}
	}

	addTop(func(i, j int) bool { return list[i].CPUPct > list[j].CPUPct }, func(p *Process) float64 { return p.CPUPct })
	addTop(func(i, j int) bool { return list[i].MemRSS > list[j].MemRSS }, func(p *Process) float64 { return p.MemRSS })
	addTop(func(i, j int) bool { return list[i].DiskRead > list[j].DiskRead }, func(p *Process) float64 { return p.DiskRead })
	addTop(func(i, j int) bool { return list[i].DiskWrite > list[j].DiskWrite }, func(p *Process) float64 { return p.DiskWrite })

	return winners
}
