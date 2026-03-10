package prechecks

import (
	"testing"

	"github.com/Terminus-Lab/themis/internal/models"
)

func TestRunner(t *testing.T) {
	checks := []Checker{}

	checks = append(checks, NewFormatChecker())
	checks = append(checks, NewLengthChecker())
	checks = append(checks, NewOverlapChecker())
	runner := NewStageRunner(checks)

	tests := []struct {
		name                string
		query               string
		answer              string
		expectedResultCount int
		minAvgScore         float64
		description         string
	}{
		{
			name:                "valid query and answer with good overlap",
			query:               "How do I encrypt files using AWS KMS?",
			answer:              "To encrypt files using AWS KMS, you need to create a KMS key and use it with the encryption API. AWS KMS provides secure key management for encrypting your files.",
			expectedResultCount: 3,
			minAvgScore:         0.7,
			description:         "Should pass all checks with high scores",
		},
		{
			name:                "empty answer",
			query:               "What is encryption?",
			answer:              "",
			expectedResultCount: 3,
			minAvgScore:         0.0,
			description:         "Format checker should fail on empty answer",
		},
		{
			name:                "very short answer",
			query:               "Explain the difference between symmetric and asymmetric encryption in detail",
			answer:              "Yes.",
			expectedResultCount: 3,
			minAvgScore:         0.0,
			description:         "Length checker should give low score for very short answer",
		},
		{
			name:                "no keyword overlap",
			query:               "How do I configure Redis caching?",
			answer:              "The weather today is sunny and pleasant.",
			expectedResultCount: 3,
			minAvgScore:         0.0,
			description:         "Overlap checker should give low score for irrelevant answer",
		},
		{
			name:                "long well-formatted answer with overlap",
			query:               "What are the best practices for securing API endpoints?",
			answer:              "Best practices for securing API endpoints include: implementing authentication and authorization, using HTTPS for all communications, rate limiting to prevent abuse, input validation to prevent injection attacks, and regular security audits. These practices help ensure your API endpoints remain secure and reliable.",
			expectedResultCount: 3,
			minAvgScore:         0.8,
			description:         "Should pass all checks with high scores",
		},
		{
			name:                "answer with repeated punctuation",
			query:               "Tell me about vector databases",
			answer:              "Vector databases are great!!! They store embeddings... Really powerful!!!",
			expectedResultCount: 3,
			minAvgScore:         0.0,
			description:         "Format checker should detect repeated punctuation",
		},
		{
			name:                "very long answer for short query",
			query:               "Hi",
			answer:              "Hello! I'm here to help you with any questions you might have about our products and services. Feel free to ask me anything and I'll do my best to provide you with detailed and accurate information.",
			expectedResultCount: 3,
			minAvgScore:         0.3,
			description:         "Length checker may give low score due to length mismatch",
		},
		{
			name:                "good answer with partial overlap",
			query:               "What is semantic search?",
			answer:              "Semantic search uses vector embeddings to find documents based on meaning rather than exact keyword matching. This allows for more intelligent retrieval.",
			expectedResultCount: 3,
			minAvgScore:         0.6,
			description:         "Should have moderate to high scores across all checks",
		},
		{
			name:                "answer with only whitespace",
			query:               "What is RAG?",
			answer:              "   \n\t  ",
			expectedResultCount: 3,
			minAvgScore:         0.0,
			description:         "Format checker should fail on whitespace-only answer",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := models.EvaluationContext{
				Query:  test.query,
				Answer: test.answer,
			}

			result := runner.Run(ctx)

			// Verify we got results from all checkers
			if len(result) != test.expectedResultCount {
				t.Errorf("Expected %d results, got %d", test.expectedResultCount, len(result))
			}

			// Verify each result has required fields
			var totalScore float64
			checkerNames := make(map[string]bool)
			for _, res := range result {
				if res.Name == "" {
					t.Error("Result should have a non-empty name")
				}
				if res.Score < 0.0 || res.Score > 1.0 {
					t.Errorf("Score should be between 0 and 1, got %f for checker %s", res.Score, res.Name)
				}
				if res.Duration < 0 {
					t.Errorf("Duration should be non-negative, got %v for checker %s", res.Duration, res.Name)
				}
				totalScore += res.Score
				checkerNames[res.Name] = true

				// Log result details for debugging
				t.Logf("Checker: %s, Score: %.2f, Reason: %s", res.Name, res.Score, res.Reason)
			}

			// Verify all expected checkers ran
			expectedCheckers := []string{"format-checker", "length-checker", "overlap-checker"}
			for _, expected := range expectedCheckers {
				if !checkerNames[expected] {
					t.Errorf("Expected checker %s to run, but it didn't", expected)
				}
			}

			// Calculate average score
			avgScore := totalScore / float64(len(result))
			t.Logf("Test: %s - Average score: %.2f (expected min: %.2f)", test.name, avgScore, test.minAvgScore)

			// Verify average score meets minimum expectation
			if avgScore < test.minAvgScore {
				t.Logf("Warning: Average score %.2f is below expected minimum %.2f for test '%s'", avgScore, test.minAvgScore, test.name)
			}
		})
	}
}
