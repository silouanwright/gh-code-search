package github

import (
	"context"
	"time"
)

// GitHubAPI defines the interface for GitHub API operations
type GitHubAPI interface {
	SearchCode(ctx context.Context, query string, opts *SearchOptions) (*SearchResults, error)
	GetFileContent(ctx context.Context, owner, repo, path, ref string) ([]byte, error)
	GetRateLimit(ctx context.Context) (*RateLimit, error)
}

// SearchOptions configures search requests
type SearchOptions struct {
	Sort           string // relevance, indexed, created, updated
	Order          string // asc, desc
	ListOptions    ListOptions
	SkipEnrichment bool // Skip fetching additional repository metadata (stars, etc)
}

// ListOptions specifies pagination options
type ListOptions struct {
	Page    int
	PerPage int
}

// SearchResults represents GitHub search results
type SearchResults struct {
	Total             *int         `json:"total_count,omitempty"`
	IncompleteResults *bool        `json:"incomplete_results,omitempty"`
	Items             []SearchItem `json:"items,omitempty"`
}

// SearchItem represents a single search result
type SearchItem struct {
	Name        *string     `json:"name,omitempty"`
	Path        *string     `json:"path,omitempty"`
	SHA         *string     `json:"sha,omitempty"`
	URL         *string     `json:"url,omitempty"`
	GitURL      *string     `json:"git_url,omitempty"`
	HTMLURL     *string     `json:"html_url,omitempty"`
	Repository  Repository  `json:"repository,omitempty"`
	Score       *float64    `json:"score,omitempty"`
	TextMatches []TextMatch `json:"text_matches,omitempty"`
}

// Repository represents a GitHub repository
type Repository struct {
	ID              *int64     `json:"id,omitempty"`
	NodeID          *string    `json:"node_id,omitempty"`
	Name            *string    `json:"name,omitempty"`
	FullName        *string    `json:"full_name,omitempty"`
	Owner           *User      `json:"owner,omitempty"`
	Private         *bool      `json:"private,omitempty"`
	HTMLURL         *string    `json:"html_url,omitempty"`
	Description     *string    `json:"description,omitempty"`
	Fork            *bool      `json:"fork,omitempty"`
	CreatedAt       *time.Time `json:"created_at,omitempty"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
	PushedAt        *time.Time `json:"pushed_at,omitempty"`
	StargazersCount *int       `json:"stargazers_count,omitempty"`
	WatchersCount   *int       `json:"watchers_count,omitempty"`
	ForksCount      *int       `json:"forks_count,omitempty"`
	Language        *string    `json:"language,omitempty"`
	DefaultBranch   *string    `json:"default_branch,omitempty"`
}

// User represents a GitHub user or organization
type User struct {
	Login     *string `json:"login,omitempty"`
	ID        *int64  `json:"id,omitempty"`
	NodeID    *string `json:"node_id,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	HTMLURL   *string `json:"html_url,omitempty"`
	Type      *string `json:"type,omitempty"`
}

// TextMatch represents highlighted text matches in search results
type TextMatch struct {
	ObjectURL  *string `json:"object_url,omitempty"`
	ObjectType *string `json:"object_type,omitempty"`
	Property   *string `json:"property,omitempty"`
	Fragment   *string `json:"fragment,omitempty"`
	Matches    []Match `json:"matches,omitempty"`
}

// Match represents a specific match within a text fragment
type Match struct {
	Text    *string `json:"text,omitempty"`
	Indices []int   `json:"indices,omitempty"`
}

// RateLimit represents GitHub API rate limiting information
type RateLimit struct {
	Limit     int       `json:"limit"`
	Remaining int       `json:"remaining"`
	Reset     time.Time `json:"reset"`
}

// Helper methods for Repository
func (r *Repository) GetOwnerLogin() string {
	if r.Owner != nil && r.Owner.Login != nil {
		return *r.Owner.Login
	}
	return ""
}

func (r *Repository) GetName() string {
	if r.Name != nil {
		return *r.Name
	}
	return ""
}

// Helper functions for pointer conversion
func IntPtr(i int) *int {
	return &i
}

func StringPtr(s string) *string {
	return &s
}

func BoolPtr(b bool) *bool {
	return &b
}

func Float64Ptr(f float64) *float64 {
	return &f
}
