package llm

import (
	"bufio"
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
	// Initialize a slice to hold the sub-questions
	var subQuestions []string

	// Create a scanner to read the response line by line
	scanner := bufio.NewScanner(strings.NewReader(response))

	// Flag to indicate when we've reached the sub-questions
	subQuestionsStarted := false

	// Iterate through each line of the response
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Check if we've reached the start of the sub-questions
		if strings.HasPrefix(line, "SUB QUESTIONS:") {
			subQuestionsStarted = true
			continue
		}

		// If we've started processing sub-questions and the line is not empty
		if subQuestionsStarted && line != "" {
			// Remove the number and period at the start of the line
			parts := strings.SplitN(line, ". ", 2)
			if len(parts) == 2 {
				subQuestions = append(subQuestions, parts[1])
			}
		}
	}

	// Check for scanning errors
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning response: %w", err)
	}

	// Check if we found any sub-questions
	if len(subQuestions) == 0 {
		return []string{}, nil
	}

	return subQuestions, nil
}

// Helper function to truncate a string
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength] + "..."
}
