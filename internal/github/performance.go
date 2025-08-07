package github

import (
	"fmt"
	"strings"
	"time"
)

// PerformanceMetrics tracks performance data for batch operations
type PerformanceMetrics struct {
	StartTime         time.Time     `json:"start_time"`
	EndTime           time.Time     `json:"end_time"`
	TotalDuration     time.Duration `json:"total_duration"`
	TotalSearches     int           `json:"total_searches"`
	SuccessfulSearches int          `json:"successful_searches"`
	FailedSearches    int           `json:"failed_searches"`
	TotalResults      int           `json:"total_results"`
	RetryCount        int           `json:"retry_count"`
	DelayTime         time.Duration `json:"total_delay_time"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	RateLimitHits     int           `json:"rate_limit_hits"`
	AbuseDetections   int           `json:"abuse_detections"`
	ServerErrors      int           `json:"server_errors"`
}

// SearchMetrics tracks metrics for individual searches
type SearchMetrics struct {
	SearchName    string        `json:"search_name"`
	Query         string        `json:"query"`
	StartTime     time.Time     `json:"start_time"`
	Duration      time.Duration `json:"duration"`
	ResultCount   int           `json:"result_count"`
	RetryCount    int           `json:"retry_count"`
	DelayTime     time.Duration `json:"delay_time"`
	ErrorType     string        `json:"error_type,omitempty"`
	Success       bool          `json:"success"`
}

// PerformanceTracker manages performance tracking for batch operations
type PerformanceTracker struct {
	metrics       *PerformanceMetrics
	searchMetrics []SearchMetrics
	currentSearch *SearchMetrics
}

// NewPerformanceTracker creates a new performance tracker
func NewPerformanceTracker() *PerformanceTracker {
	return &PerformanceTracker{
		metrics: &PerformanceMetrics{
			StartTime: time.Now(),
		},
		searchMetrics: make([]SearchMetrics, 0),
	}
}

// StartBatch initializes tracking for a batch operation
func (pt *PerformanceTracker) StartBatch(totalSearches int) {
	pt.metrics.StartTime = time.Now()
	pt.metrics.TotalSearches = totalSearches
	pt.searchMetrics = make([]SearchMetrics, 0, totalSearches)
}

// StartSearch begins tracking an individual search
func (pt *PerformanceTracker) StartSearch(name, query string) {
	pt.currentSearch = &SearchMetrics{
		SearchName: name,
		Query:      query,
		StartTime:  time.Now(),
	}
}

// EndSearch completes tracking for the current search
func (pt *PerformanceTracker) EndSearch(resultCount int, err error) {
	if pt.currentSearch == nil {
		return
	}

	pt.currentSearch.Duration = time.Since(pt.currentSearch.StartTime)
	pt.currentSearch.ResultCount = resultCount
	pt.currentSearch.Success = err == nil

	if err != nil {
		pt.currentSearch.ErrorType = classifyError(err)
		pt.metrics.FailedSearches++

		// Track specific error types
		switch pt.currentSearch.ErrorType {
		case "rate_limit":
			pt.metrics.RateLimitHits++
		case "abuse_detection":
			pt.metrics.AbuseDetections++
		case "server_error":
			pt.metrics.ServerErrors++
		}
	} else {
		pt.metrics.SuccessfulSearches++
		pt.metrics.TotalResults += resultCount
	}

	pt.searchMetrics = append(pt.searchMetrics, *pt.currentSearch)
	pt.currentSearch = nil
}

// RecordRetry tracks retry attempts
func (pt *PerformanceTracker) RecordRetry() {
	if pt.currentSearch != nil {
		pt.currentSearch.RetryCount++
	}
	pt.metrics.RetryCount++
}

// RecordDelay tracks delay time
func (pt *PerformanceTracker) RecordDelay(delay time.Duration) {
	if pt.currentSearch != nil {
		pt.currentSearch.DelayTime += delay
	}
	pt.metrics.DelayTime += delay
}

// EndBatch completes batch operation tracking
func (pt *PerformanceTracker) EndBatch() {
	pt.metrics.EndTime = time.Now()
	pt.metrics.TotalDuration = pt.metrics.EndTime.Sub(pt.metrics.StartTime)

	// Calculate average response time (excluding delays)
	if pt.metrics.SuccessfulSearches > 0 {
		totalResponseTime := time.Duration(0)
		for _, search := range pt.searchMetrics {
			if search.Success {
				// Subtract delay time to get actual response time
				responseTime := search.Duration - search.DelayTime
				totalResponseTime += responseTime
			}
		}
		pt.metrics.AverageResponseTime = totalResponseTime / time.Duration(pt.metrics.SuccessfulSearches)
	}
}

// GetMetrics returns the current performance metrics
func (pt *PerformanceTracker) GetMetrics() *PerformanceMetrics {
	return pt.metrics
}

// GetSearchMetrics returns metrics for individual searches
func (pt *PerformanceTracker) GetSearchMetrics() []SearchMetrics {
	return pt.searchMetrics
}

// GenerateReport creates a human-readable performance report
func (pt *PerformanceTracker) GenerateReport() string {
	m := pt.metrics

	report := fmt.Sprintf(`🚀 **Batch Operation Performance Report**

⏱️  **Timing**:
  • Total duration: %s
  • Average response time: %s
  • Total delay time: %s
  • Actual work time: %s

📊 **Results**:
  • Total searches: %d
  • Successful: %d (%.1f%%)
  • Failed: %d (%.1f%%)
  • Total results found: %d
  • Average results per search: %.1f

🔄 **Reliability**:
  • Total retries: %d
  • Rate limit hits: %d
  • Abuse detections: %d
  • Server errors: %d

⚡ **Performance Insights**:`,
		m.TotalDuration,
		m.AverageResponseTime,
		m.DelayTime,
		m.TotalDuration-m.DelayTime,
		m.TotalSearches,
		m.SuccessfulSearches, float64(m.SuccessfulSearches)/float64(m.TotalSearches)*100,
		m.FailedSearches, float64(m.FailedSearches)/float64(m.TotalSearches)*100,
		m.TotalResults,
		float64(m.TotalResults)/float64(max(m.SuccessfulSearches, 1)),
		m.RetryCount,
		m.RateLimitHits,
		m.AbuseDetections,
		m.ServerErrors)

	// Add performance insights
	if m.AverageResponseTime > 2*time.Second {
		report += "\n  • ⚠️ Slower than expected response times - consider smaller batch sizes"
	} else {
		report += "\n  • ✅ Good response times"
	}

	if m.RateLimitHits > 0 {
		report += fmt.Sprintf("\n  • ⚠️ Hit rate limits %d times - consider adding delays", m.RateLimitHits)
	}

	if m.AbuseDetections > 0 {
		report += fmt.Sprintf("\n  • ⚠️ Triggered abuse detection %d times - reduce request frequency", m.AbuseDetections)
	}

	if m.RetryCount > m.TotalSearches {
		report += "\n  • ⚠️ High retry rate - check network connection and GitHub status"
	}

	successRate := float64(m.SuccessfulSearches) / float64(m.TotalSearches) * 100
	if successRate < 90 {
		report += "\n  • ⚠️ Low success rate - investigate common failure patterns"
	} else {
		report += "\n  • ✅ High success rate"
	}

	// Add recommendations
	report += "\n\n💡 **Recommendations**:"

	if m.DelayTime > m.TotalDuration/2 {
		report += "\n  • Consider optimizing delays - they account for most of the execution time"
	}

	if m.RateLimitHits > 0 || m.AbuseDetections > 0 {
		report += "\n  • Implement longer delays between searches"
		report += "\n  • Use more specific filters to reduce API load"
	}

	if m.AverageResponseTime > time.Second && m.TotalResults/max(m.SuccessfulSearches, 1) < 10 {
		report += "\n  • Consider increasing max_results per search to get more data per API call"
	}

	return report
}

// GenerateDetailedReport includes individual search metrics
func (pt *PerformanceTracker) GenerateDetailedReport() string {
	report := pt.GenerateReport()

	if len(pt.searchMetrics) > 0 {
		report += "\n\n📋 **Individual Search Performance**:\n"

		for i, search := range pt.searchMetrics {
			status := "✅"
			if !search.Success {
				status = "❌"
			}

			report += fmt.Sprintf("  %d. %s %s\n", i+1, status, search.SearchName)
			report += fmt.Sprintf("     Duration: %s | Results: %d",
				search.Duration, search.ResultCount)

			if search.RetryCount > 0 {
				report += fmt.Sprintf(" | Retries: %d", search.RetryCount)
			}

			if search.DelayTime > 0 {
				report += fmt.Sprintf(" | Delays: %s", search.DelayTime)
			}

			if !search.Success {
				report += fmt.Sprintf(" | Error: %s", search.ErrorType)
			}

			report += "\n"
		}
	}

	return report
}

// Helper functions

// classifyError categorizes errors for metrics tracking
func classifyError(err error) string {
	if err == nil {
		return ""
	}

	errStr := err.Error()

	if _, ok := err.(*RateLimitError); ok {
		return "rate_limit"
	}

	if _, ok := err.(*AbuseRateLimitError); ok {
		return "abuse_detection"
	}

	if _, ok := err.(*AuthenticationError); ok {
		return "authentication"
	}

	if _, ok := err.(*AuthorizationError); ok {
		return "authorization"
	}

	if _, ok := err.(*NotFoundError); ok {
		return "not_found"
	}

	if _, ok := err.(*ValidationError); ok {
		return "validation"
	}

	// Check for server errors by string matching
	errLower := strings.ToLower(errStr)
	if strings.Contains(errLower, "500") || strings.Contains(errLower, "502") ||
	   strings.Contains(errLower, "503") || strings.Contains(errLower, "504") ||
	   strings.Contains(errLower, "server error") {
		return "server_error"
	}

	return "other"
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
