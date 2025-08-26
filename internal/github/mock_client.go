package github

import (
	"context"
	"fmt"
	"time"
)

// MockClient implements GitHubAPI for testing
type MockClient struct {
	SearchResults          map[string]*SearchResults
	PaginatedSearchResults map[string]map[int]*SearchResults // query -> page -> results
	FileContents           map[string][]byte
	Errors                 map[string]error
	CallLog                []MockCall
	RateLimits             map[string]*RateLimit
}

// MockCall represents a logged API call for verification
type MockCall struct {
	Method string
	Args   []interface{}
	Time   time.Time
}

// NewMockClient creates a new mock GitHub API client
func NewMockClient() *MockClient {
	return &MockClient{
		SearchResults:          make(map[string]*SearchResults),
		PaginatedSearchResults: make(map[string]map[int]*SearchResults),
		FileContents:           make(map[string][]byte),
		Errors:                 make(map[string]error),
		CallLog:                make([]MockCall, 0),
		RateLimits:             make(map[string]*RateLimit),
	}
}

// SearchCode implements GitHubAPI.SearchCode for testing
func (m *MockClient) SearchCode(ctx context.Context, query string, opts *SearchOptions) (*SearchResults, error) {
	m.logCall("SearchCode", query, opts)

	// Check for configured errors first
	if err, exists := m.Errors["SearchCode"]; exists {
		return nil, err
	}

	// Check for paginated results first (for testing pagination)
	if paginatedResults, exists := m.PaginatedSearchResults[query]; exists {
		page := 1
		if opts != nil && opts.ListOptions.Page > 0 {
			page = opts.ListOptions.Page
		}
		if pageResults, pageExists := paginatedResults[page]; pageExists {
			return pageResults, nil
		}
		// If specific page not found, return empty results
		return &SearchResults{
			Total: IntPtr(0),
			Items: []SearchItem{},
		}, nil
	}

	// Return configured single-page results
	if results, exists := m.SearchResults[query]; exists {
		return results, nil
	}

	// Default to empty results
	return &SearchResults{
		Total: IntPtr(0),
		Items: []SearchItem{},
	}, nil
}

// GetFileContent implements GitHubAPI.GetFileContent for testing
func (m *MockClient) GetFileContent(ctx context.Context, owner, repo, path, ref string) ([]byte, error) {
	key := fmt.Sprintf("%s/%s/%s@%s", owner, repo, path, ref)
	m.logCall("GetFileContent", owner, repo, path, ref)

	// Check for configured errors
	if err, exists := m.Errors["GetFileContent"]; exists {
		return nil, err
	}

	// Return configured content
	if content, exists := m.FileContents[key]; exists {
		return content, nil
	}

	// Default to empty content
	return []byte{}, nil
}

// GetRateLimit implements GitHubAPI.GetRateLimit for testing
func (m *MockClient) GetRateLimit(ctx context.Context) (*RateLimit, error) {
	m.logCall("GetRateLimit")

	// Check for configured errors
	if err, exists := m.Errors["GetRateLimit"]; exists {
		return nil, err
	}

	// Return configured rate limit or default
	if limit, exists := m.RateLimits["default"]; exists {
		return limit, nil
	}

	// Default rate limit
	return &RateLimit{
		Limit:     30,
		Remaining: 29,
		Reset:     time.Now().Add(time.Hour),
	}, nil
}

// Helper methods for test setup

// SetSearchResults configures mock search results for a query
func (m *MockClient) SetSearchResults(query string, results *SearchResults) {
	m.SearchResults[query] = results
}

// SetPaginatedSearchResults configures mock paginated search results for testing pagination
func (m *MockClient) SetPaginatedSearchResults(query string, pageResults map[int]*SearchResults) {
	m.PaginatedSearchResults[query] = pageResults
}

// SetFileContent configures mock file content
func (m *MockClient) SetFileContent(owner, repo, path, ref string, content []byte) {
	key := fmt.Sprintf("%s/%s/%s@%s", owner, repo, path, ref)
	m.FileContents[key] = content
}

// SetError configures an error for a specific method
func (m *MockClient) SetError(method string, err error) {
	m.Errors[method] = err
}

// SetRateLimit configures mock rate limit information
func (m *MockClient) SetRateLimit(limit *RateLimit) {
	m.RateLimits["default"] = limit
}

// GetCallLog returns the logged API calls for verification
func (m *MockClient) GetCallLog() []MockCall {
	return m.CallLog
}

// ClearCallLog clears the call log
func (m *MockClient) ClearCallLog() {
	m.CallLog = make([]MockCall, 0)
}

// GetLastCall returns the most recent API call
func (m *MockClient) GetLastCall() *MockCall {
	if len(m.CallLog) == 0 {
		return nil
	}
	return &m.CallLog[len(m.CallLog)-1]
}

// GetCallCount returns the number of calls to a specific method
func (m *MockClient) GetCallCount(method string) int {
	count := 0
	for _, call := range m.CallLog {
		if call.Method == method {
			count++
		}
	}
	return count
}

// VerifyCall checks if a specific method was called with expected arguments
func (m *MockClient) VerifyCall(method string, args ...interface{}) bool {
	for _, call := range m.CallLog {
		if call.Method == method {
			if len(args) == 0 {
				return true // Just verify method was called
			}
			if len(call.Args) >= len(args) {
				match := true
				for i, expectedArg := range args {
					if call.Args[i] != expectedArg {
						match = false
						break
					}
				}
				if match {
					return true
				}
			}
		}
	}
	return false
}

// GetAllCalls returns all logged API calls for verification
func (m *MockClient) GetAllCalls() []MockCall {
	return m.CallLog
}

// Reset clears all mock data and call logs
func (m *MockClient) Reset() {
	m.SearchResults = make(map[string]*SearchResults)
	m.PaginatedSearchResults = make(map[string]map[int]*SearchResults)
	m.FileContents = make(map[string][]byte)
	m.Errors = make(map[string]error)
	m.CallLog = make([]MockCall, 0)
	m.RateLimits = make(map[string]*RateLimit)
}

// Private helper to log API calls
func (m *MockClient) logCall(method string, args ...interface{}) {
	call := MockCall{
		Method: method,
		Args:   args,
		Time:   time.Now(),
	}
	m.CallLog = append(m.CallLog, call)
}

// Helper functions for creating test data

// CreateTestSearchResults creates sample search results for testing
func CreateTestSearchResults(totalCount int, items ...SearchItem) *SearchResults {
	return &SearchResults{
		Total: IntPtr(totalCount),
		Items: items,
	}
}

// CreateTestSearchItem creates a sample search item for testing
func CreateTestSearchItem(repoName, path, content string) SearchItem {
	return SearchItem{
		Name:    StringPtr(extractFileName(path)),
		Path:    StringPtr(path),
		HTMLURL: StringPtr(fmt.Sprintf("https://github.com/%s/blob/main/%s", repoName, path)),
		Repository: Repository{
			FullName:        StringPtr(repoName),
			HTMLURL:         StringPtr(fmt.Sprintf("https://github.com/%s", repoName)),
			StargazersCount: IntPtr(1000),
		},
		TextMatches: []TextMatch{
			{
				Fragment: StringPtr(content),
				Property: StringPtr("content"),
			},
		},
	}
}

// Helper to extract filename from path
func extractFileName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}
