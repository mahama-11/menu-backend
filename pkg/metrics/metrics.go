package metrics

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type httpMetric struct {
	Count         int64
	DurationMsSum int64
}

var (
	mu              sync.Mutex
	httpMetrics     = map[string]*httpMetric{}
	businessMetrics = map[string]int64{}
)

func RecordHTTPRequest(method, path string, status int, duration time.Duration) {
	key := fmt.Sprintf("%s|%s|%d", method, path, status)
	mu.Lock()
	defer mu.Unlock()
	item := httpMetrics[key]
	if item == nil {
		item = &httpMetric{}
		httpMetrics[key] = item
	}
	item.Count++
	item.DurationMsSum += duration.Milliseconds()
}

func IncBusinessCounter(name string) {
	mu.Lock()
	defer mu.Unlock()
	businessMetrics[name]++
}

func RenderPrometheus(namespace, subsystem string) string {
	prefix := namespace
	if subsystem != "" {
		prefix = prefix + "_" + subsystem
	}
	if prefix == "" {
		prefix = "menu_service"
	}

	mu.Lock()
	defer mu.Unlock()

	var lines []string
	lines = append(lines,
		fmt.Sprintf("# HELP %s_http_requests_total Total HTTP requests", prefix),
		fmt.Sprintf("# TYPE %s_http_requests_total counter", prefix),
	)

	httpKeys := make([]string, 0, len(httpMetrics))
	for key := range httpMetrics {
		httpKeys = append(httpKeys, key)
	}
	sort.Strings(httpKeys)
	for _, key := range httpKeys {
		parts := strings.Split(key, "|")
		if len(parts) != 3 {
			continue
		}
		item := httpMetrics[key]
		lines = append(lines,
			fmt.Sprintf(`%s_http_requests_total{method="%s",path="%s",status="%s"} %d`, prefix, parts[0], escapeLabel(parts[1]), parts[2], item.Count),
			fmt.Sprintf(`%s_http_request_duration_ms_sum{method="%s",path="%s",status="%s"} %d`, prefix, parts[0], escapeLabel(parts[1]), parts[2], item.DurationMsSum),
		)
	}

	lines = append(lines,
		fmt.Sprintf("# HELP %s_business_events_total Total business events", prefix),
		fmt.Sprintf("# TYPE %s_business_events_total counter", prefix),
	)
	bizKeys := make([]string, 0, len(businessMetrics))
	for key := range businessMetrics {
		bizKeys = append(bizKeys, key)
	}
	sort.Strings(bizKeys)
	for _, key := range bizKeys {
		lines = append(lines, fmt.Sprintf(`%s_business_events_total{name="%s"} %d`, prefix, escapeLabel(key), businessMetrics[key]))
	}

	return strings.Join(lines, "\n") + "\n"
}

func escapeLabel(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}
