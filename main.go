package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/deziss/proc-sentry/collector"
	"github.com/deziss/proc-sentry/config"
	"github.com/deziss/proc-sentry/internal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	userMap := internal.LoadUserMap("/etc/passwd")

	pc, err := collector.NewProcCollector(cfg, userMap)
	if err != nil {
		log.Fatalf("Failed to initialize collector: %v", err)
	}

	prometheus.MustRegister(pc)
	prometheus.MustRegister(*pc.ScrapeDuration, pc.ScrapeErrors, pc.ScrapePanics)

	log.Printf("Starting proc-sentry on :%s (TOP_N=%d, DiskIO=%v, Ports=%v, PROCFS=%s, Interval=%s)",
		cfg.Port, cfg.TopN, cfg.EnableDiskIO, cfg.EnablePorts, cfg.ProcFSPath, cfg.UpdateInterval)

	// Start background collection loop
	stop := make(chan struct{})
	go pc.Run(stop)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	// Graceful shutdown: honor SIGTERM, drain in-flight scrapes, then exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v, shutting down gracefully...", sig)

		// Stop the collector loop
		close(stop)

		// Give in-flight HTTP requests 5 seconds to finish
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Error starting server: %v", err)
	}

	log.Println("proc-sentry stopped")
}
