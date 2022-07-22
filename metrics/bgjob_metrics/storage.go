package bgjob_metrics

import (
	"time"

	"github.com/integration-system/isp-kit/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type Storage struct {
	duration           *prometheus.SummaryVec
	dlqCount           *prometheus.CounterVec
	retryCount         *prometheus.CounterVec
	successCount       *prometheus.CounterVec
	internalErrorCount prometheus.Counter
}

func NewStorage(reg *metrics.Registry) *Storage {
	s := &Storage{
		duration: prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Subsystem:  "bgjob",
			Name:       "execute_duration_ms",
			Help:       "The latency of execution single job from queue",
			Objectives: metrics.DefaultObjectives,
		}, []string{"queue", "job_type"}),
		dlqCount: prometheus.NewCounterVec(prometheus.CounterOpts{
			Subsystem: "bgjob",
			Name:      "execute_dlq_count",
			Help:      "Count of jobs moved to DLQ",
		}, []string{"queue", "job_type"}),
		retryCount: prometheus.NewCounterVec(prometheus.CounterOpts{
			Subsystem: "bgjob",
			Name:      "execute_retry_count",
			Help:      "Count of retried jobs",
		}, []string{"queue", "job_type"}),
		successCount: prometheus.NewCounterVec(prometheus.CounterOpts{
			Subsystem: "bgjob",
			Name:      "execute_success_count",
			Help:      "Count of successful jobs",
		}, []string{"queue", "job_type"}),
		internalErrorCount: prometheus.NewCounter(prometheus.CounterOpts{
			Subsystem: "bgjob",
			Name:      "worker_error_count",
			Help:      "Count of internal worker errors",
		}),
	}
	s.duration = reg.GetOrRegister(s.duration).(*prometheus.SummaryVec)
	s.retryCount = reg.GetOrRegister(s.retryCount).(*prometheus.CounterVec)
	s.dlqCount = reg.GetOrRegister(s.dlqCount).(*prometheus.CounterVec)
	s.successCount = reg.GetOrRegister(s.successCount).(*prometheus.CounterVec)
	s.internalErrorCount = reg.GetOrRegister(s.internalErrorCount).(prometheus.Counter)
	return s
}

func (c *Storage) ObserveExecuteDuration(queue string, jobType string, t time.Duration) {
	c.duration.WithLabelValues(queue, jobType).Observe(float64(t.Milliseconds()))
}

func (c *Storage) IncRetryCount(queue string, jobType string) {
	c.retryCount.WithLabelValues(queue, jobType).Inc()
}

func (c *Storage) IncDlqCount(queue string, jobType string) {
	c.dlqCount.WithLabelValues(queue, jobType).Inc()
}

func (c *Storage) IncSuccessCount(queue string, jobType string) {
	c.successCount.WithLabelValues(queue, jobType).Inc()
}

func (c *Storage) IncInternalErrorCount() {
	c.internalErrorCount.Inc()
}
