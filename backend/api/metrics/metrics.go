// Package metrics defines the Prometheus metrics this backend exposes on
// /metrics. Metrics are scoped to the parts of the architecture that are
// actually interesting to observe: HTTP traffic, the WebSocket hub's
// backpressure behavior (bounded channel, non-blocking send), Kafka publish
// retries, and Redis cache-aside hit rate. No MongoDB/Kafka broker-level
// metrics here — those are better covered by a dedicated exporter than by
// app code, and weren't worth the extra infra for this project's scope.
package metrics

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests processed, labeled by route, method and status code.",
		},
		[]string{"route", "method", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds, labeled by route and method.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"route", "method"},
	)

	// WSConnectionsActive is a per-node gauge: each pod only knows about the
	// WebSocket connections it's holding locally, matching the per-node
	// consumer group design (see kafka.NewStatusManager / CLAUDE.md).
	WSConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "chat_ws_connections_active",
			Help: "WebSocket chat connections currently held by this node.",
		},
	)

	// ChatMessagesDelivered/Dropped instrument hub.go's DeliverMessage
	// select/default: Send is a buffered channel (cap 256) with a
	// non-blocking send so one stalled client can't block the hub. Dropped
	// means that buffer was full and the message was discarded — previously
	// only visible as a log line.
	ChatMessagesDelivered = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "chat_messages_delivered_total",
			Help: "Chat messages successfully handed to a locally-connected client's send channel.",
		},
	)

	ChatMessagesDropped = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "chat_messages_dropped_total",
			Help: "Chat messages dropped because the recipient's send channel was full.",
		},
	)

	KafkaPublishRetries = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_publish_retries_total",
			Help: "Retry attempts made while publishing a chat message to Kafka.",
		},
	)

	KafkaPublishFailures = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_publish_failures_total",
			Help: "Chat messages that failed to publish to Kafka after exhausting all retries.",
		},
	)

	// CacheHits/Misses are labeled by the endpoint the cache-aside read lives
	// in (post_controller.go, user_controller.go), so hit rate can be
	// compared across endpoints instead of one blended number.
	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Redis cache-aside reads that found a cached value.",
		},
		[]string{"cache"},
	)

	CacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Redis cache-aside reads that missed and fell through to MongoDB.",
		},
		[]string{"cache"},
	)
)

// HTTPMiddleware records HTTPRequestsTotal/HTTPRequestDuration for every
// request. Registered once in server.NewHTTPServer.
func HTTPMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		route := c.Route().Path
		status := strconv.Itoa(c.Response().StatusCode())
		HTTPRequestsTotal.WithLabelValues(route, c.Method(), status).Inc()
		HTTPRequestDuration.WithLabelValues(route, c.Method()).Observe(time.Since(start).Seconds())

		return err
	}
}
