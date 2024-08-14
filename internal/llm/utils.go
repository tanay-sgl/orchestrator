package llm

import (
	"fmt"
	"strings"
)

func FormatSubQuestionAnswers(subQuestionAnswers []string) string {
	var result strings.Builder

	for i, entry := range subQuestionAnswers {
		// Add a numbered header for each sub-question and answer pair
		result.WriteString(fmt.Sprintf("%d. %s\n", i+1, entry))

		// Add a separator between entries, except for the last one
		if i < len(subQuestionAnswers)-1 {
			result.WriteString("\n---\n\n")
		}
	}

	return result.String()
}

func ParseSubQuestions(response string) ([]string, error) {
	// Find the opening and closing brackets
	start := strings.Index(response, "[")
	end := strings.LastIndex(response, "]")

	if start == -1 || end == -1 || start >= end {
		return []string{}, nil
	}

	// Extract the content between the brackets
	content := response[start+1 : end]

	// Split the content by commas
	questions := strings.Split(content, ",")

	// Trim whitespace and remove quotes from each question
	for i, q := range questions {
		q = strings.TrimSpace(q)
		q = strings.Trim(q, "\"")
		questions[i] = q
	}

	// Remove any empty questions
	var nonEmptyQuestions []string
	for _, q := range questions {
		if q != "" {
			nonEmptyQuestions = append(nonEmptyQuestions, q)
		}
	}

	return nonEmptyQuestions, nil
}

// Helper function to truncate a string
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength] + "..."
}
