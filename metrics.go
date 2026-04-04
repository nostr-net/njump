package main

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var metricRegion string

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "njump_http_requests_total",
			Help: "Total HTTP requests handled by njump.",
		},
		[]string{"region", "domain", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "njump_http_request_duration_seconds",
			Help:    "End-to-end HTTP request duration.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"region", "domain", "path", "status"},
	)

	panicTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "njump_panics_total",
			Help: "Recovered panic count by recovery point.",
		},
		[]string{"region", "domain", "source"},
	)

	timeoutTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "njump_timeouts_total",
			Help: "Timed out HTTP requests.",
		},
		[]string{"region", "domain", "path"},
	)

	queueOutcomeTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "njump_queue_outcomes_total",
			Help: "Queue middleware outcomes.",
		},
		[]string{"region", "domain", "outcome"},
	)

	relayDiscoveryRunsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "njump_relay_discovery_runs_total",
			Help: "Relay discovery attempts by outcome.",
		},
		[]string{"region", "outcome"},
	)

	relayDiscoveryCandidateCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "njump_relay_discovery_candidates",
			Help: "Number of relay candidates discovered from each source.",
		},
		[]string{"region", "source"},
	)

	relayPoolSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "njump_relay_pool_size",
			Help: "Configured relay count by relay pool.",
		},
		[]string{"region", "pool"},
	)

	buildInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "njump_build_info",
			Help: "Build metadata for the running njump binary.",
		},
		[]string{"region", "build_ts"},
	)
)

func init() {
	metricRegion = os.Getenv("REGION")
	if metricRegion == "" {
		metricRegion = "unknown"
	}
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		panicTotal,
		timeoutTotal,
		queueOutcomeTotal,
		relayDiscoveryRunsTotal,
		relayDiscoveryCandidateCount,
		relayPoolSize,
		buildInfo,
	)
}

func recordRelayDiscoveryRun(outcome string) {
	relayDiscoveryRunsTotal.WithLabelValues(metricRegion, outcome).Inc()
}

func setRelayDiscoveryCandidateCount(source string, count int) {
	label := source
	if label == "" {
		label = "unknown"
	}
	if len(label) > 128 {
		label = label[:128]
	}
	relayDiscoveryCandidateCount.WithLabelValues(metricRegion, label).Set(float64(count))
}

func setRelayPoolSize(pool string, size int) {
	relayPoolSize.WithLabelValues(metricRegion, pool).Set(float64(size))
}

func metricsHandler() http.Handler {
	return promhttp.Handler()
}

func metricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		mw := &metricsResponseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(mw, r)

		region := metricRegion
		domain := metricsDomainLabel(r)
		path := metricsPathLabel(r.URL.Path)
		status := strconv.Itoa(mw.status)
		httpRequestsTotal.WithLabelValues(region, domain, path, status).Inc()
		httpRequestDuration.WithLabelValues(region, domain, path, status).Observe(time.Since(start).Seconds())
	}
}

func setBuildInfoMetric(buildTS string) {
	if buildTS == "" {
		buildTS = "dev"
	}
	buildInfo.Reset()
	buildInfo.WithLabelValues(metricRegion, buildTS).Set(1)
}

func recordPanic(r *http.Request, source string) {
	panicTotal.WithLabelValues(metricRegion, metricsDomainLabel(r), source).Inc()
}

func recordTimeout(r *http.Request) {
	timeoutTotal.WithLabelValues(metricRegion, metricsDomainLabel(r), metricsPathLabel(r.URL.Path)).Inc()
}

func recordQueueOutcome(r *http.Request, outcome string) {
	queueOutcomeTotal.WithLabelValues(metricRegion, metricsDomainLabel(r), outcome).Inc()
}

type metricsResponseWriter struct {
	http.ResponseWriter
	status int
}

func (mw *metricsResponseWriter) WriteHeader(status int) {
	mw.status = status
	mw.ResponseWriter.WriteHeader(status)
}

func (mw *metricsResponseWriter) Write(b []byte) (int, error) {
	return mw.ResponseWriter.Write(b)
}

func metricsPathLabel(path string) string {
	switch {
	case path == "/":
		return "homepage"
	case path == "/about":
		return "about"
	case path == "/metrics":
		return "metrics"
	case path == "/robots.txt":
		return "robots"
	case strings.HasPrefix(path, "/njump/static/"):
		return "static"
	case strings.HasPrefix(path, "/njump/image/"), strings.HasPrefix(path, "/image/"):
		return "image"
	case strings.HasPrefix(path, "/njump/proxy/"), strings.HasPrefix(path, "/proxy/"):
		return "proxy"
	case strings.HasPrefix(path, "/services/oembed"):
		return "oembed"
	case strings.HasPrefix(path, "/embed/"):
		return "embed"
	case strings.HasPrefix(path, "/r/"):
		return "relay"
	case strings.HasPrefix(path, "/random"):
		return "random"
	case strings.HasPrefix(path, "/e/"), strings.HasPrefix(path, "/p/"):
		return "legacy_redirect"
	default:
		return "resource"
	}
}

func metricsDomainLabel(r *http.Request) string {
	if r == nil {
		return s.Domain
	}
	if domain, ok := r.Context().Value("domain").(string); ok && domain != "" {
		return domain
	}

	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.TrimPrefix(strings.ToLower(host), "www.")
	if host == "" {
		return s.Domain
	}
	return host
}

func isMetricsPath(path string) bool {
	return path == "/metrics" || strings.HasPrefix(path, "/metrics/")
}
