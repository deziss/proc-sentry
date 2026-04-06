package collector

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/deziss/proc-sentry/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

var metricLabels = []string{"pid", "user", "command", "runtime", "rank", "hostname", "container_id", "ports", "full_path"}

// ProcCollector implements prometheus.Collector to avoid Reset() race conditions.
// Metrics are built on-demand during Collect() from a snapshot updated by the background loop.
type ProcCollector struct {
	cfg     *config.Config
	userMap map[string]string
	sysFS   procfs.FS

	mu          sync.RWMutex
	lastSysTime float64
	procTicks   map[int]float64
	snapshot    []*Process // latest collected processes

	// Descriptors
	cpuDesc       *prometheus.Desc
	memDesc       *prometheus.Desc
	diskReadDesc  *prometheus.Desc
	diskWriteDesc *prometheus.Desc
	totalDesc     *prometheus.Desc

	// Self-monitoring (these are safe to use concurrently)
	ScrapeDuration *prometheus.Histogram
	ScrapeErrors   prometheus.Counter
	ScrapePanics   prometheus.Counter
}

func NewProcCollector(cfg *config.Config, userMap map[string]string) (*ProcCollector, error) {
	sysFS, err := procfs.NewFS(cfg.ProcFSPath)
	if err != nil {
		return nil, fmt.Errorf("initializing procfs at %s: %w", cfg.ProcFSPath, err)
	}

	duration := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "proc_scrape_duration_seconds",
		Help:    "Scrape duration",
		Buckets: prometheus.DefBuckets,
	})

	c := &ProcCollector{
		cfg:       cfg,
		userMap:   userMap,
		sysFS:     sysFS,
		procTicks: make(map[int]float64),

		cpuDesc:       prometheus.NewDesc("proc_process_top_cpu_percent", "Top processes by CPU percentage (per core scale, 100% = 1 core)", metricLabels, nil),
		memDesc:       prometheus.NewDesc("proc_process_top_memory_bytes", "Top processes by RSS Memory in bytes", metricLabels, nil),
		diskReadDesc:  prometheus.NewDesc("proc_process_top_disk_read_bytes", "Top processes by Disk Read bytes", metricLabels, nil),
		diskWriteDesc: prometheus.NewDesc("proc_process_top_disk_write_bytes", "Top processes by Disk Write bytes", metricLabels, nil),
		totalDesc:     prometheus.NewDesc("proc_processes_scraped_total", "Total processes scraped", []string{"runtime"}, nil),

		ScrapeDuration: &duration,
		ScrapeErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "proc_scrape_errors_total",
			Help: "Total scrape errors",
		}),
		ScrapePanics: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "proc_scrape_panics_total",
			Help: "Total panics recovered during scrape",
		}),
	}

	return c, nil
}

// Describe implements prometheus.Collector.
func (c *ProcCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.cpuDesc
	ch <- c.memDesc
	ch <- c.diskReadDesc
	ch <- c.diskWriteDesc
	ch <- c.totalDesc
}

// Collect implements prometheus.Collector. It reads the latest snapshot and emits metrics.
// This is called by Prometheus during scrape — no Reset() needed.
func (c *ProcCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.RLock()
	procs := c.snapshot
	c.mu.RUnlock()

	if len(procs) == 0 {
		return
	}

	// Work on a copy of the slice to avoid sorting the shared snapshot
	list := make([]*Process, len(procs))
	copy(list, procs)

	winners := SelectTopN(list, c.cfg.TopN)

	if c.cfg.EnablePorts && len(winners) > 0 {
		for _, p := range winners {
			if p.ProcfsProc != nil {
				p.Ports = ResolvePortsSafely(p.ProcfsProc, c.cfg.ProcFSPath)
			}
		}
	}

	ch <- prometheus.MustNewConstMetric(c.totalDesc, prometheus.GaugeValue, float64(len(list)), "host")

	emitTop := func(desc *prometheus.Desc, less func(i, j int) bool, val func(*Process) float64) {
		sort.Slice(list, less)
		for i := 0; i < c.cfg.TopN && i < len(list); i++ {
			p := list[i]
			v := val(p)
			if v == 0 {
				continue
			}
			ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v,
				strconv.Itoa(p.PID), p.User, p.Command, p.Runtime, strconv.Itoa(i+1),
				c.cfg.Hostname, p.ContainerID, p.Ports, p.FullPath,
			)
		}
	}

	emitTop(c.cpuDesc, func(i, j int) bool { return list[i].CPUPct > list[j].CPUPct }, func(p *Process) float64 { return p.CPUPct })
	emitTop(c.memDesc, func(i, j int) bool { return list[i].MemRSS > list[j].MemRSS }, func(p *Process) float64 { return p.MemRSS })
	emitTop(c.diskReadDesc, func(i, j int) bool { return list[i].DiskRead > list[j].DiskRead }, func(p *Process) float64 { return p.DiskRead })
	emitTop(c.diskWriteDesc, func(i, j int) bool { return list[i].DiskWrite > list[j].DiskWrite }, func(p *Process) float64 { return p.DiskWrite })
}

// Run starts the background collection loop. It blocks until the stop channel is closed.
func (c *ProcCollector) Run(stop <-chan struct{}) {
	ticker := time.NewTicker(c.cfg.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			c.collectSafe()
		}
	}
}

func (c *ProcCollector) collectSafe() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("CRITICAL: Panic recovered in collect loop: %v", r)
			c.ScrapePanics.Inc()
		}
	}()

	start := time.Now()
	err := c.collect()
	(*c.ScrapeDuration).Observe(time.Since(start).Seconds())

	if err != nil {
		c.ScrapeErrors.Inc()
		log.Printf("Error during collection: %v", err)
	}
}

func (c *ProcCollector) collect() error {
	stat, err := c.sysFS.Stat()
	if err != nil {
		return fmt.Errorf("could not read system stat: %w", err)
	}

	currSysTime := stat.CPUTotal.Idle + stat.CPUTotal.User + stat.CPUTotal.System +
		stat.CPUTotal.Iowait + stat.CPUTotal.IRQ + stat.CPUTotal.SoftIRQ +
		stat.CPUTotal.Steal + stat.CPUTotal.Nice

	c.mu.Lock()
	if c.lastSysTime == 0 {
		c.lastSysTime = currSysTime
	}
	sysDiff := currSysTime - c.lastSysTime
	if sysDiff <= 0 {
		sysDiff = 1
	}
	prevTicks := c.procTicks
	c.mu.Unlock()

	numCPUs := float64(len(stat.CPU))
	if numCPUs == 0 {
		numCPUs = 1
	}

	procs, err := c.sysFS.AllProcs()
	if err != nil {
		return fmt.Errorf("could not read processes: %w", err)
	}

	var parsed []*Process
	newTicks := make(map[int]float64, len(procs))

	for _, p := range procs {
		proc, err := ReadProcess(p, c.userMap, c.cfg.EnableDiskIO)
		if err != nil {
			continue
		}

		if prev, ok := prevTicks[proc.PID]; ok {
			diff := proc.Ticks - prev
			if diff >= 0 {
				proc.CPUPct = (diff / sysDiff) * 100 * numCPUs
			}
		}

		newTicks[proc.PID] = proc.Ticks
		parsed = append(parsed, proc)
	}

	c.mu.Lock()
	c.procTicks = newTicks
	c.lastSysTime = currSysTime
	c.snapshot = parsed
	c.mu.Unlock()

	return nil
}
