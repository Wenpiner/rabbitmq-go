package common

import (
	"fmt"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

// Metrics 测试指标
type Metrics struct {
	StartTime         time.Time
	MessagesSent      atomic.Int64
	MessagesReceived  atomic.Int64
	MessagesFailed    atomic.Int64
	ReconnectCount    atomic.Int64
	ErrorCount        atomic.Int64
	LastError         atomic.Value
	CurrentGoroutines atomic.Int64
	MemoryAllocMB     atomic.Int64
}

// NewMetrics 创建指标收集器
func NewMetrics() *Metrics {
	m := &Metrics{
		StartTime: time.Now(),
	}
	
	// 启动指标收集
	go m.collectSystemMetrics()
	
	return m
}

// collectSystemMetrics 收集系统指标
func (m *Metrics) collectSystemMetrics() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		
		m.CurrentGoroutines.Store(int64(runtime.NumGoroutine()))
		m.MemoryAllocMB.Store(int64(memStats.Alloc / 1024 / 1024))
	}
}

// RecordSent 记录发送
func (m *Metrics) RecordSent() {
	m.MessagesSent.Add(1)
}

// RecordReceived 记录接收
func (m *Metrics) RecordReceived() {
	m.MessagesReceived.Add(1)
}

// RecordFailed 记录失败
func (m *Metrics) RecordFailed() {
	m.MessagesFailed.Add(1)
}

// RecordError 记录错误
func (m *Metrics) RecordError(err error) {
	m.ErrorCount.Add(1)
	m.LastError.Store(err.Error())
}

// RecordReconnect 记录重连
func (m *Metrics) RecordReconnect() {
	m.ReconnectCount.Add(1)
}

// GetStats 获取统计信息
func (m *Metrics) GetStats() map[string]interface{} {
	duration := time.Since(m.StartTime)
	sent := m.MessagesSent.Load()
	received := m.MessagesReceived.Load()
	
	stats := map[string]interface{}{
		"uptime_seconds":      duration.Seconds(),
		"messages_sent":       sent,
		"messages_received":   received,
		"messages_failed":     m.MessagesFailed.Load(),
		"reconnect_count":     m.ReconnectCount.Load(),
		"error_count":         m.ErrorCount.Load(),
		"current_goroutines":  m.CurrentGoroutines.Load(),
		"memory_alloc_mb":     m.MemoryAllocMB.Load(),
		"send_rate_per_sec":   float64(sent) / duration.Seconds(),
		"receive_rate_per_sec": float64(received) / duration.Seconds(),
	}
	
	if lastErr := m.LastError.Load(); lastErr != nil {
		stats["last_error"] = lastErr
	}
	
	return stats
}

// PrintStats 打印统计信息
func (m *Metrics) PrintStats() {
	stats := m.GetStats()
	fmt.Println("\n=== 测试统计 ===")
	fmt.Printf("运行时间: %.2f 秒\n", stats["uptime_seconds"])
	fmt.Printf("发送消息: %d (%.2f msg/s)\n", stats["messages_sent"], stats["send_rate_per_sec"])
	fmt.Printf("接收消息: %d (%.2f msg/s)\n", stats["messages_received"], stats["receive_rate_per_sec"])
	fmt.Printf("失败消息: %d\n", stats["messages_failed"])
	fmt.Printf("重连次数: %d\n", stats["reconnect_count"])
	fmt.Printf("错误次数: %d\n", stats["error_count"])
	fmt.Printf("Goroutine: %d\n", stats["current_goroutines"])
	fmt.Printf("内存使用: %d MB\n", stats["memory_alloc_mb"])
	if lastErr, ok := stats["last_error"]; ok {
		fmt.Printf("最后错误: %s\n", lastErr)
	}
	fmt.Println("================")
}

// ServeMetrics 启动 HTTP 指标服务
func (m *Metrics) ServeMetrics(addr string) error {
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		stats := m.GetStats()
		w.Header().Set("Content-Type", "text/plain")
		
		// Prometheus 格式
		fmt.Fprintf(w, "# HELP messages_sent_total Total messages sent\n")
		fmt.Fprintf(w, "# TYPE messages_sent_total counter\n")
		fmt.Fprintf(w, "messages_sent_total %d\n", stats["messages_sent"])
		
		fmt.Fprintf(w, "# HELP messages_received_total Total messages received\n")
		fmt.Fprintf(w, "# TYPE messages_received_total counter\n")
		fmt.Fprintf(w, "messages_received_total %d\n", stats["messages_received"])
		
		fmt.Fprintf(w, "# HELP messages_failed_total Total messages failed\n")
		fmt.Fprintf(w, "# TYPE messages_failed_total counter\n")
		fmt.Fprintf(w, "messages_failed_total %d\n", stats["messages_failed"])
		
		fmt.Fprintf(w, "# HELP reconnect_count_total Total reconnections\n")
		fmt.Fprintf(w, "# TYPE reconnect_count_total counter\n")
		fmt.Fprintf(w, "reconnect_count_total %d\n", stats["reconnect_count"])
		
		fmt.Fprintf(w, "# HELP goroutines Current number of goroutines\n")
		fmt.Fprintf(w, "# TYPE goroutines gauge\n")
		fmt.Fprintf(w, "goroutines %d\n", stats["current_goroutines"])
		
		fmt.Fprintf(w, "# HELP memory_alloc_mb Memory allocated in MB\n")
		fmt.Fprintf(w, "# TYPE memory_alloc_mb gauge\n")
		fmt.Fprintf(w, "memory_alloc_mb %d\n", stats["memory_alloc_mb"])
	})
	
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})
	
	return http.ListenAndServe(addr, nil)
}

