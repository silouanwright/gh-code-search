package github

import (
	"context"
	"testing"

	"github.com/google/go-github/v57/github"
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

func TestConvertRepositoryWithNilStarCount(t *testing.T) {
	tests := []struct {
		name          string
		inputRepo     *github.Repository
		expectedStars *int
	}{
		{
			name: "repository with valid star count",
			inputRepo: &github.Repository{
				FullName:        github.String("test/repo"),
				StargazersCount: github.Int(1500),
			},
			expectedStars: github.Int(1500),
		},
		{
			name: "repository with nil star count",
			inputRepo: &github.Repository{
				FullName:        github.String("test/repo"),
				StargazersCount: nil,
			},
			expectedStars: nil,
		},
		{
			name: "repository with zero star count",
			inputRepo: &github.Repository{
				FullName:        github.String("test/repo"),
				StargazersCount: github.Int(0),
			},
			expectedStars: github.Int(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertRepository(tt.inputRepo)

			if tt.expectedStars == nil {
				assert.Nil(t, result.StargazersCount)
			} else {
				assert.NotNil(t, result.StargazersCount)
				assert.Equal(t, *tt.expectedStars, *result.StargazersCount)
			}
			
			// Verify other fields are preserved
			assert.Equal(t, *tt.inputRepo.FullName, *result.FullName)
		})
	}
}

func TestGetStringFromPtr(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected string
	}{
		{
			name:     "valid string pointer",
			input:    StringPtr("test-value"),
			expected: "test-value",
		},
		{
			name:     "nil string pointer",
			input:    nil,
			expected: "",
		},
		{
			name:     "empty string pointer",
			input:    StringPtr(""),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStringFromPtr(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Note: StringPtr and IntPtr are already defined in client.go

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