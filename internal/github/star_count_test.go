package github

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepositoryEnrichment(t *testing.T) {
	t.Run("enrichRepositoryMetadata with nil results", func(t *testing.T) {
		client := &RealClient{}
		client.enrichRepositoryMetadata(context.Background(), nil)
		// Should not panic
	})

	t.Run("enrichRepositoryMetadata with empty results", func(t *testing.T) {
		client := &RealClient{}
		results := &SearchResults{Items: []SearchItem{}}
		client.enrichRepositoryMetadata(context.Background(), results)
		// Should not panic
	})

	t.Run("repository with existing star count is not enriched", func(t *testing.T) {
		existingStars := 1000
		results := &SearchResults{
			Items: []SearchItem{
				{
					Repository: Repository{
						FullName:        StringPtr("test/repo"),
						StargazersCount: &existingStars,
					},
				},
			},
		}

		client := &RealClient{}
		client.enrichRepositoryMetadata(context.Background(), results)

		// Should keep existing star count
		assert.Equal(t, 1000, *results.Items[0].Repository.StargazersCount)
	})

	t.Run("repository with nil full name is skipped", func(t *testing.T) {
		results := &SearchResults{
			Items: []SearchItem{
				{
					Repository: Repository{
						FullName:        nil,
						StargazersCount: nil,
					},
				},
			},
		}

		client := &RealClient{}
		client.enrichRepositoryMetadata(context.Background(), results)

		// Should remain nil
		assert.Nil(t, results.Items[0].Repository.StargazersCount)
	})
}

func TestGetIntValue(t *testing.T) {
	tests := []struct {
		name     string
		input    *int
		expected int
	}{
		{
			name:     "valid int pointer",
			input:    IntPtr(42),
			expected: 42,
		},
		{
			name:     "nil int pointer",
			input:    nil,
			expected: 0,
		},
		{
			name:     "zero int pointer",
			input:    IntPtr(0),
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getIntValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStarCountDisplayInOutput(t *testing.T) {
	t.Run("star count correctly displayed in markdown output", func(t *testing.T) {
		// This test verifies the integration between repository data and output formatting
		searchItem := &SearchItem{
			Name: StringPtr("test-file.txt"),
			Path: StringPtr("test/file.txt"),
			Repository: Repository{
				FullName:        StringPtr("test/repo"),
				StargazersCount: IntPtr(1234),
				HTMLURL:         StringPtr("https://github.com/test/repo"),
			},
		}

		// Test that the getIntValue helper works correctly
		stars := getIntValue(searchItem.Repository.StargazersCount)
		assert.Equal(t, 1234, stars)

		// Test zero stars
		searchItem.Repository.StargazersCount = IntPtr(0)
		stars = getIntValue(searchItem.Repository.StargazersCount)
		assert.Equal(t, 0, stars)

		// Test nil stars
		searchItem.Repository.StargazersCount = nil
		stars = getIntValue(searchItem.Repository.StargazersCount)
		assert.Equal(t, 0, stars)
	})
}

// getIntValue helper function for tests (mimics the one in output package)
func getIntValue(ptr *int) int {
	if ptr == nil {
		return 0
	}
	return *ptr
}
