package github

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// TestNewPerformanceTracker tests creation of performance tracker
func TestNewPerformanceTracker(t *testing.T) {
	pt := NewPerformanceTracker()
	
	if pt.metrics == nil {
		t.Error("Expected metrics to be initialized")
	}
	
	if pt.searchMetrics == nil {
		t.Error("Expected searchMetrics to be initialized")
	}
	
	if pt.currentSearch != nil {
		t.Error("Expected currentSearch to be nil initially")
	}
}

// TestStartBatch tests batch initialization
func TestStartBatch(t *testing.T) {
	pt := NewPerformanceTracker()
	totalSearches := 5
	
	pt.StartBatch(totalSearches)
	
	if pt.metrics.TotalSearches != totalSearches {
		t.Errorf("Expected TotalSearches to be %d, got %d", totalSearches, pt.metrics.TotalSearches)
	}
	
	if cap(pt.searchMetrics) != totalSearches {
		t.Errorf("Expected searchMetrics capacity to be %d, got %d", totalSearches, cap(pt.searchMetrics))
	}
	
	if pt.metrics.StartTime.IsZero() {
		t.Error("Expected StartTime to be set")
	}
}

// TestStartEndSearch tests individual search tracking
func TestStartEndSearch(t *testing.T) {
	pt := NewPerformanceTracker()
	pt.StartBatch(1)
	
	searchName := "test search"
	query := "test query"
	
	// Start search
	pt.StartSearch(searchName, query)
	
	if pt.currentSearch == nil {
		t.Fatal("Expected currentSearch to be set")
	}
	
	if pt.currentSearch.SearchName != searchName {
		t.Errorf("Expected SearchName to be %s, got %s", searchName, pt.currentSearch.SearchName)
	}
	
	if pt.currentSearch.Query != query {
		t.Errorf("Expected Query to be %s, got %s", query, pt.currentSearch.Query)
	}
	
	if pt.currentSearch.StartTime.IsZero() {
		t.Error("Expected StartTime to be set")
	}
	
	// Wait a bit to ensure duration is measurable
	time.Sleep(10 * time.Millisecond)
	
	// End search successfully
	resultCount := 42
	pt.EndSearch(resultCount, nil)
	
	if pt.currentSearch != nil {
		t.Error("Expected currentSearch to be nil after EndSearch")
	}
	
	if len(pt.searchMetrics) != 1 {
		t.Errorf("Expected 1 search metric, got %d", len(pt.searchMetrics))
	}
	
	metric := pt.searchMetrics[0]
	if metric.SearchName != searchName {
		t.Errorf("Expected SearchName to be %s, got %s", searchName, metric.SearchName)
	}
	
	if metric.ResultCount != resultCount {
		t.Errorf("Expected ResultCount to be %d, got %d", resultCount, metric.ResultCount)
	}
	
	if !metric.Success {
		t.Error("Expected Success to be true")
	}
	
	if metric.Duration == 0 {
		t.Error("Expected Duration to be > 0")
	}
	
	// Check that metrics were updated
	if pt.metrics.SuccessfulSearches != 1 {
		t.Errorf("Expected SuccessfulSearches to be 1, got %d", pt.metrics.SuccessfulSearches)
	}
	
	if pt.metrics.TotalResults != resultCount {
		t.Errorf("Expected TotalResults to be %d, got %d", resultCount, pt.metrics.TotalResults)
	}
}

// TestEndSearchWithError tests error handling in search tracking
func TestEndSearchWithError(t *testing.T) {
	pt := NewPerformanceTracker()
	pt.StartBatch(1)
	pt.StartSearch("test search", "test query")
	
	// End search with error
	err := &RateLimitError{Message: "Rate limit exceeded"}
	pt.EndSearch(0, err)
	
	if len(pt.searchMetrics) != 1 {
		t.Errorf("Expected 1 search metric, got %d", len(pt.searchMetrics))
	}
	
	metric := pt.searchMetrics[0]
	if metric.Success {
		t.Error("Expected Success to be false")
	}
	
	if metric.ErrorType != "rate_limit" {
		t.Errorf("Expected ErrorType to be 'rate_limit', got %s", metric.ErrorType)
	}
	
	// Check that error counters were updated
	if pt.metrics.FailedSearches != 1 {
		t.Errorf("Expected FailedSearches to be 1, got %d", pt.metrics.FailedSearches)
	}
	
	if pt.metrics.RateLimitHits != 1 {
		t.Errorf("Expected RateLimitHits to be 1, got %d", pt.metrics.RateLimitHits)
	}
}

// TestRecordRetry tests retry tracking
func TestRecordRetry(t *testing.T) {
	pt := NewPerformanceTracker()
	pt.StartBatch(1)
	pt.StartSearch("test search", "test query")
	
	// Record some retries
	pt.RecordRetry()
	pt.RecordRetry()
	
	if pt.currentSearch.RetryCount != 2 {
		t.Errorf("Expected current search RetryCount to be 2, got %d", pt.currentSearch.RetryCount)
	}
	
	if pt.metrics.RetryCount != 2 {
		t.Errorf("Expected total RetryCount to be 2, got %d", pt.metrics.RetryCount)
	}
}

// TestRecordDelay tests delay tracking
func TestRecordDelay(t *testing.T) {
	pt := NewPerformanceTracker()
	pt.StartBatch(1)
	pt.StartSearch("test search", "test query")
	
	delay1 := 100 * time.Millisecond
	delay2 := 200 * time.Millisecond
	
	pt.RecordDelay(delay1)
	pt.RecordDelay(delay2)
	
	expectedTotal := delay1 + delay2
	
	if pt.currentSearch.DelayTime != expectedTotal {
		t.Errorf("Expected current search DelayTime to be %v, got %v", expectedTotal, pt.currentSearch.DelayTime)
	}
	
	if pt.metrics.DelayTime != expectedTotal {
		t.Errorf("Expected total DelayTime to be %v, got %v", expectedTotal, pt.metrics.DelayTime)
	}
}

// TestEndBatch tests batch completion and metric calculation
func TestEndBatch(t *testing.T) {
	pt := NewPerformanceTracker()
	pt.StartBatch(2)
	
	// Add two successful searches
	pt.StartSearch("search1", "query1")
	time.Sleep(10 * time.Millisecond)
	pt.EndSearch(10, nil)
	
	pt.StartSearch("search2", "query2")
	time.Sleep(15 * time.Millisecond)
	pt.EndSearch(15, nil)
	
	pt.EndBatch()
	
	if pt.metrics.EndTime.IsZero() {
		t.Error("Expected EndTime to be set")
	}
	
	if pt.metrics.TotalDuration == 0 {
		t.Error("Expected TotalDuration to be > 0")
	}
	
	if pt.metrics.AverageResponseTime == 0 {
		t.Error("Expected AverageResponseTime to be > 0")
	}
	
	// Verify final counts
	if pt.metrics.SuccessfulSearches != 2 {
		t.Errorf("Expected SuccessfulSearches to be 2, got %d", pt.metrics.SuccessfulSearches)
	}
	
	if pt.metrics.TotalResults != 25 {
		t.Errorf("Expected TotalResults to be 25, got %d", pt.metrics.TotalResults)
	}
}

// TestClassifyError tests error classification
func TestClassifyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"nil error", nil, ""},
		{"rate limit", &RateLimitError{}, "rate_limit"},
		{"abuse detection", &AbuseRateLimitError{}, "abuse_detection"},
		{"authentication", &AuthenticationError{}, "authentication"},
		{"authorization", &AuthorizationError{}, "authorization"},
		{"not found", &NotFoundError{}, "not_found"},
		{"validation", &ValidationError{}, "validation"},
		{"server error 500", errors.New("500 Internal Server Error"), "server_error"},
		{"server error 502", errors.New("502 Bad Gateway"), "server_error"},
		{"server error 503", errors.New("503 Service Unavailable"), "server_error"},
		{"server error 504", errors.New("504 Gateway Timeout"), "server_error"},
		{"generic error", errors.New("unknown error"), "other"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestGenerateReport tests report generation
func TestGenerateReport(t *testing.T) {
	pt := NewPerformanceTracker()
	pt.StartBatch(2)
	
	// Simulate batch operation
	pt.StartSearch("search1", "query1")
	pt.EndSearch(10, nil)
	
	pt.StartSearch("search2", "query2")
	pt.RecordRetry()
	pt.EndSearch(0, &RateLimitError{Message: "Rate limit"})
	
	pt.EndBatch()
	
	report := pt.GenerateReport()
	
	// Check that report contains expected sections
	expectedSections := []string{
		"Batch Operation Performance Report",
		"Timing",
		"Results",
		"Reliability",
		"Performance Insights",
	}
	
	for _, section := range expectedSections {
		if !strings.Contains(report, section) {
			t.Errorf("Expected report to contain '%s' section", section)
		}
	}
	
	// Check specific metrics are included
	if !strings.Contains(report, "Total searches: 2") {
		t.Error("Expected report to include total searches")
	}
	
	if !strings.Contains(report, "Successful: 1") {
		t.Error("Expected report to include successful searches")
	}
	
	if !strings.Contains(report, "Failed: 1") {
		t.Error("Expected report to include failed searches")
	}
	
	if !strings.Contains(report, "Rate limit hits: 1") {
		t.Error("Expected report to include rate limit hits")
	}
}

// TestGenerateDetailedReport tests detailed report generation
func TestGenerateDetailedReport(t *testing.T) {
	pt := NewPerformanceTracker()
	pt.StartBatch(1)
	
	pt.StartSearch("test search", "test query")
	pt.EndSearch(5, nil)
	
	pt.EndBatch()
	
	report := pt.GenerateDetailedReport()
	
	// Should include individual search details
	if !strings.Contains(report, "Individual Search Performance") {
		t.Error("Expected detailed report to include individual search performance")
	}
	
	if !strings.Contains(report, "test search") {
		t.Error("Expected detailed report to include search name")
	}
	
	if !strings.Contains(report, "Results: 5") {
		t.Error("Expected detailed report to include result count")
	}
}

// TestPerformanceMetricsAccuracy tests accuracy of performance calculations
func TestPerformanceMetricsAccuracy(t *testing.T) {
	pt := NewPerformanceTracker()
	pt.StartBatch(3)
	
	// First search: success with 10 results
	pt.StartSearch("search1", "query1")
	time.Sleep(20 * time.Millisecond)
	pt.EndSearch(10, nil)
	
	// Second search: rate limited, then success with 5 results
	pt.StartSearch("search2", "query2")
	pt.RecordRetry() // First attempt fails
	pt.RecordDelay(50 * time.Millisecond)
	time.Sleep(10 * time.Millisecond)
	pt.EndSearch(5, nil)
	
	// Third search: fails with server error
	pt.StartSearch("search3", "query3")
	pt.RecordRetry()
	pt.RecordRetry()
	time.Sleep(5 * time.Millisecond)
	pt.EndSearch(0, errors.New("500 Server Error"))
	
	pt.EndBatch()
	
	metrics := pt.GetMetrics()
	
	// Verify accuracy
	if metrics.TotalSearches != 3 {
		t.Errorf("Expected TotalSearches=3, got %d", metrics.TotalSearches)
	}
	
	if metrics.SuccessfulSearches != 2 {
		t.Errorf("Expected SuccessfulSearches=2, got %d", metrics.SuccessfulSearches)
	}
	
	if metrics.FailedSearches != 1 {
		t.Errorf("Expected FailedSearches=1, got %d", metrics.FailedSearches)
	}
	
	if metrics.TotalResults != 15 {
		t.Errorf("Expected TotalResults=15, got %d", metrics.TotalResults)
	}
	
	if metrics.RetryCount != 3 {
		t.Errorf("Expected RetryCount=3, got %d", metrics.RetryCount)
	}
	
	if metrics.DelayTime != 50*time.Millisecond {
		t.Errorf("Expected DelayTime=50ms, got %v", metrics.DelayTime)
	}
	
	if metrics.ServerErrors != 1 {
		t.Errorf("Expected ServerErrors=1, got %d", metrics.ServerErrors)
	}
	
	// Average response time should exclude the failed search
	if metrics.AverageResponseTime == 0 {
		t.Error("Expected AverageResponseTime > 0")
	}
}

// BenchmarkPerformanceTracker benchmarks performance tracking overhead
func BenchmarkPerformanceTracker(b *testing.B) {
	pt := NewPerformanceTracker()
	pt.StartBatch(b.N)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pt.StartSearch("search", "query")
		pt.EndSearch(10, nil)
	}
	pt.EndBatch()
}

// BenchmarkReportGeneration benchmarks report generation
func BenchmarkReportGeneration(b *testing.B) {
	// Setup tracker with data
	pt := NewPerformanceTracker()
	pt.StartBatch(100)
	
	for i := 0; i < 100; i++ {
		pt.StartSearch("search", "query")
		if i%10 == 0 {
			pt.RecordRetry()
		}
		pt.EndSearch(5, nil)
	}
	pt.EndBatch()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pt.GenerateReport()
	}
}