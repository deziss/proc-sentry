package collector

import (
	"testing"
)

func TestSelectTopN_Basic(t *testing.T) {
	procs := []*Process{
		{PID: 1, CPUPct: 90, MemRSS: 100},
		{PID: 2, CPUPct: 50, MemRSS: 500},
		{PID: 3, CPUPct: 10, MemRSS: 900},
		{PID: 4, CPUPct: 5, MemRSS: 10},
	}

	winners := SelectTopN(procs, 2)

	// PID 1 (top CPU), PID 2 (2nd CPU), PID 3 (top mem), PID 2 (2nd mem) = {1,2,3}
	if _, ok := winners[1]; !ok {
		t.Error("PID 1 should be a winner (top CPU)")
	}
	if _, ok := winners[2]; !ok {
		t.Error("PID 2 should be a winner (2nd CPU, 2nd mem)")
	}
	if _, ok := winners[3]; !ok {
		t.Error("PID 3 should be a winner (top mem)")
	}
}

func TestSelectTopN_ZeroValuesExcluded(t *testing.T) {
	procs := []*Process{
		{PID: 1, CPUPct: 0, MemRSS: 0},
		{PID: 2, CPUPct: 10, MemRSS: 100},
	}

	winners := SelectTopN(procs, 5)

	if _, ok := winners[1]; ok {
		t.Error("PID 1 with all zeros should not be a winner")
	}
	if _, ok := winners[2]; !ok {
		t.Error("PID 2 should be a winner")
	}
}

func TestSelectTopN_LargerThanList(t *testing.T) {
	procs := []*Process{
		{PID: 1, CPUPct: 50, MemRSS: 200},
		{PID: 2, CPUPct: 30, MemRSS: 100},
	}

	winners := SelectTopN(procs, 100)

	if len(winners) != 2 {
		t.Errorf("expected 2 winners, got %d", len(winners))
	}
}

func TestSelectTopN_Empty(t *testing.T) {
	winners := SelectTopN(nil, 10)
	if len(winners) != 0 {
		t.Errorf("expected 0 winners for nil input, got %d", len(winners))
	}
}

func TestSelectTopN_DiskIO(t *testing.T) {
	procs := []*Process{
		{PID: 1, CPUPct: 1, MemRSS: 1, DiskRead: 0, DiskWrite: 9000},
		{PID: 2, CPUPct: 1, MemRSS: 1, DiskRead: 9000, DiskWrite: 0},
		{PID: 3, CPUPct: 99, MemRSS: 99},
	}

	winners := SelectTopN(procs, 1)

	// PID 3 = top CPU + top mem, PID 1 = top disk write, PID 2 = top disk read
	if len(winners) != 3 {
		t.Errorf("expected 3 winners, got %d", len(winners))
	}
}
