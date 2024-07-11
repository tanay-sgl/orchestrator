package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gabriel-vasile/mimetype"
	"github.com/ledongthuc/pdf"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/unidoc/unioffice/document"
)

func ReadTextFile(filename string) (string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func IdentifyFileType(fileBytes []byte) (string, error) {
	// Detect MIME type
	mtype := mimetype.Detect(fileBytes)
	mimeString := mtype.String()

	// Check for PDF signature as a fallback
	if bytes.HasPrefix(fileBytes, []byte("%PDF-")) {
		return ".pdf", nil
	}

	switch {
	case strings.HasPrefix(mimeString, "application/pdf"):
		fmt.Printf("\nDetected PDF file\n")
		return ".pdf", nil
	case strings.HasPrefix(mimeString, "text/plain"):
		// Check if it looks like Markdown
		if looksLikeMarkdown(fileBytes) {
			fmt.Printf("\nDetected Markdown file\n")
			return ".md", nil
		}
		fmt.Printf("\nDetected text file\n")
		return ".txt", nil
	case strings.HasPrefix(mimeString, "application/vnd.openxmlformats-officedocument.wordprocessingml.document"):
		fmt.Printf("\nDetected Word file DOCX\n")
		return ".docx", nil
	case strings.HasPrefix(mimeString, "application/msword"):
		fmt.Printf("\nDetected Word file DOC\n")
		return ".doc", nil
	case strings.HasPrefix(mimeString, "text/markdown"):
		fmt.Printf("\nDetected Markdown file\n")
		return ".md", nil
	default:
		fmt.Printf("\nUnknown or unsupported file type: %s\n", mimeString)
		return "", errors.New("unknown or unsupported file type: " + mimeString)
	}
}

func looksLikeMarkdown(content []byte) bool {
	// Simple heuristic: check for common Markdown syntax
	mdPatterns := []string{"# ", "## ", "- ", "* ", "```", "---", "[", "]("}
	for _, pattern := range mdPatterns {
		if bytes.Contains(content, []byte(pattern)) {
			return true
		}
	}
	return false
}
func writeToFile(filename string, content string) {
	// Add timestamp to filename to avoid overwriting
	timestamp := time.Now().Format("20060102_150405")
	filename = fmt.Sprintf("%s_%s", timestamp, filename)

	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		log.Printf("Error writing to file %s: %v", filename, err)
	} else {
		log.Printf("Successfully wrote to file: %s", filename)
	}
}

func ExtractTextFromPDF(rs io.ReadSeeker) (string, error) {
	conf := model.NewDefaultConfiguration()

	tempDir, err := os.MkdirTemp("", "pdf-extract-")
	if err != nil {
		return "", fmt.Errorf("error creating temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	err = api.ExtractContent(rs, tempDir, "extracted", nil, conf)
	if err != nil {
		return "", fmt.Errorf("error extracting content from PDF: %v", err)
	}

	files, err := os.ReadDir(tempDir)
	if err != nil {
		return "", fmt.Errorf("error reading temp directory: %v", err)
	}

	var extractedFile string
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "extracted") {
			extractedFile = filepath.Join(tempDir, file.Name())
			break
		}
	}

	if extractedFile == "" {
		return "", fmt.Errorf("no extracted file found in temp directory")
	}

	content, err := os.ReadFile(extractedFile)
	if err != nil {
		return "", fmt.Errorf("error reading extracted content from %s: %v", extractedFile, err)
	}

	extractedText := extractTextFromPDFContent(string(content))
	cleanedText := removeNonPrintableCharacters(extractedText)
	normalizedText := normalizeWhitespace(cleanedText)

	return normalizedText, nil
}

// Helper function to get the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func extractTextFromPDFContent(content string) string {
	// Regular expression to find text between angle brackets (which often contain hex-encoded text)
	re := regexp.MustCompile(`<([^>]+)>`)
	matches := re.FindAllStringSubmatch(content, -1)

	var text strings.Builder
	for _, match := range matches {
		if len(match) > 1 {
			decodedText := decodeHexString(match[1])
			text.WriteString(decodedText)
			text.WriteString(" ")
		}
	}

	return text.String()
}

func decodeHexString(hexString string) string {
	var text strings.Builder
	for i := 0; i < len(hexString); i += 2 {
		if i+1 < len(hexString) {
			charCode, _ := strconv.ParseUint(hexString[i:i+2], 16, 8)
			if unicode.IsPrint(rune(charCode)) {
				text.WriteRune(rune(charCode))
			}
		}
	}
	return text.String()
}

func extractTextFromPDFFile(filePath string) (string, error) {
	// Open the PDF file
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("error opening PDF file: %v", err)
	}
	defer f.Close()

	// Get file size
	fileInfo, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("error getting file info: %v", err)
	}
	fileSize := fileInfo.Size()

	// Create a new PDF reader
	pdfReader, err := pdf.NewReader(f, fileSize)
	if err != nil {
		return "", fmt.Errorf("error creating PDF reader: %v", err)
	}

	var text strings.Builder
	for pageNum := 1; pageNum <= pdfReader.NumPage(); pageNum++ {
		page := pdfReader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		pageText, err := page.GetPlainText(nil)
		if err != nil {
			return "", fmt.Errorf("error extracting text from page %d: %v", pageNum, err)
		}
		text.WriteString(pageText)
	}

	return text.String(), nil
}

func ExtractTextFromDOCX(fileBytes []byte) (string, error) {
	tmpFile, err := os.CreateTemp("", "temp*.docx")
	if err != nil {
		return "", fmt.Errorf("error creating temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(fileBytes); err != nil {
		return "", fmt.Errorf("error writing to temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("error closing temp file: %v", err)
	}

	doc, err := document.Open(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("error opening DOCX: %v", err)
	}

	var text strings.Builder
	for _, para := range doc.Paragraphs() {
		for _, run := range para.Runs() {
			text.WriteString(run.Text())
		}
		text.WriteString("\n")
	}

	return ProcessText(text.String()), nil
}

func SplitStringIntoStringArray(text string, chunkSize int) []string {
	//fmt.Printf("SplitStringIntoStringArray\n")
	var chunks []string
	var currentChunk strings.Builder

	words := strings.Fields(text)
	for _, word := range words {
		if currentChunk.Len()+len(word)+1 > chunkSize {
			chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
			currentChunk.Reset()
		}
		currentChunk.WriteString(word)
		currentChunk.WriteString(" ")
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
	}

	return chunks
}

func normalizeWhitespace(input string) string {
	fmt.Printf("normalizeWhitespace\n")
	// Replace multiple spaces with a single space
	spaceNormalized := regexp.MustCompile(`\s+`).ReplaceAllString(input, " ")
	// Ensure single newline between paragraphs
	return regexp.MustCompile(`\n\s*\n`).ReplaceAllString(spaceNormalized, "\n\n")
}

func removeNonPrintableCharacters(input string) string {
	//fmt.Printf("removeNonPrintableCharacters\n")
	return strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, input)
}

func IdentifyAndReplaceCommonProblematicCharacters(input string) string {
	//fmt.Printf("IdentifyAndReplaceCommonProblematicCharacters\n")
	// Define a set of "safe" characters
	safeSet := &unicode.RangeTable{
		R16: []unicode.Range16{
			{Lo: 0x0020, Hi: 0x007E, Stride: 1}, // Basic Latin (printable ASCII)
			{Lo: 0x00A0, Hi: 0x00FF, Stride: 1}, // Latin-1 Supplement
		},
		R32: []unicode.Range32{
			{Lo: 0x0100, Hi: 0x017F, Stride: 1}, // Latin Extended-A
			{Lo: 0x0180, Hi: 0x024F, Stride: 1}, // Latin Extended-B
			{Lo: 0x2000, Hi: 0x206F, Stride: 1}, // General Punctuation
			{Lo: 0x2070, Hi: 0x209F, Stride: 1}, // Superscripts and Subscripts
			{Lo: 0x20A0, Hi: 0x20CF, Stride: 1}, // Currency Symbols
			{Lo: 0x2100, Hi: 0x214F, Stride: 1}, // Letterlike Symbols
			{Lo: 0x2150, Hi: 0x218F, Stride: 1}, // Number Forms
			{Lo: 0x2190, Hi: 0x21FF, Stride: 1}, // Arrows
			{Lo: 0x2200, Hi: 0x22FF, Stride: 1}, // Mathematical Operators
		},
	}

	// Replace common problematic characters
	replacer := strings.NewReplacer(
		"\u0000", "", // Null character
		"\ufffd", "", // Replacement character
		"\u200b", "", // Zero width space
		"\u200c", "", // Zero width non-joiner
		"\u200d", "", // Zero width joiner
		"\u00A0", " ", // Non-breaking space
		"\u2028", "\n", // Line separator
		"\u2029", "\n", // Paragraph separator
	)
	cleaned := replacer.Replace(input)

	// Remove any remaining characters not in the safe set
	cleaned = strings.Map(func(r rune) rune {
		if unicode.Is(safeSet, r) || unicode.IsSpace(r) {
			return r
		}
		return -1
	}, cleaned)
	//fmt.Printf("IdentifyAndReplaceCommonProblematicCharacters: %s\n", cleaned)
	// Normalize remaining whitespace
	re := regexp.MustCompile(`\s+`)
	cleaned = re.ReplaceAllString(cleaned, " ")

	return strings.TrimSpace(cleaned)
}

func ProcessText(rawText string) string {
	cleanedText := IdentifyAndReplaceCommonProblematicCharacters(rawText)
	textWithoutNonPrintable := removeNonPrintableCharacters(cleanedText)
	normalizedText := normalizeWhitespace(textWithoutNonPrintable)
	return normalizedText
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
		return nil, fmt.Errorf("no sub-questions found in the response")
	}

	return subQuestions, nil
}

func ParseDataSources(response string) ([]string, error) {
	// Trim any whitespace from the response
	response = strings.TrimSpace(response)

	// Check if the response is empty
	if response == "" {
		return nil, fmt.Errorf("empty response")
	}

	// Check if the response is "NA"
	if response == "NA" {
		return []string{"NA"}, nil
	}

	// Split the response by comma
	sources := strings.Split(response, ",")

	// Validate and clean each source
	validSources := make([]string, 0, len(sources))
	for _, source := range sources {
		// Trim whitespace
		source = strings.TrimSpace(source)

		// Validate the source
		switch source {
		case "documents", "sql", "default":
			validSources = append(validSources, source)
		default:
			return nil, fmt.Errorf("invalid data source: %s", source)
		}
	}

	// Check if we have at least one valid source
	if len(validSources) == 0 {
		return nil, fmt.Errorf("no valid data sources found")
	}

	return validSources, nil
}

func SanitizeAndParseSQLQuery(response string) (string, error) {
	//fmt.Printf(response)
	// First, let's extract the SQL query from the response
	sqlRegex := regexp.MustCompile(`(?i)SQL:\s*(.+)`)
	matches := sqlRegex.FindStringSubmatch(response)
	if len(matches) < 2 {
		return "", fmt.Errorf("no SQL query found in the response")
	}
	query := matches[1]

	// Trim any whitespace and remove any trailing semicolon
	query = strings.TrimSpace(query)
	query = strings.TrimSuffix(query, ";")

	// List of disallowed keywords (case-insensitive)
	disallowedKeywords := []string{
		"DROP", "DELETE", "TRUNCATE", "ALTER", "CREATE", "INSERT", "UPDATE",
		"GRANT", "REVOKE", "UNION", "--", "/*", "*/", "EXEC", "EXECUTE",
	}

	// Check for disallowed keywords
	lowerQuery := strings.ToLower(query)
	for _, keyword := range disallowedKeywords {
		if strings.Contains(lowerQuery, strings.ToLower(keyword)) {
			return "", fmt.Errorf("disallowed keyword found: %s", keyword)
		}
	}

	// Validate that the query starts with SELECT
	if !strings.HasPrefix(lowerQuery, "select") {
		return "", fmt.Errorf("query must start with SELECT")
	}

	// Basic structure validation
	// This is a simple check and might need to be expanded based on your specific needs
	if !strings.Contains(lowerQuery, "from") {
		return "", fmt.Errorf("invalid query structure: missing FROM clause")
	}

	return query, nil
}

func ParseRelevantData(relevantData RelevantData) (string, error) {
	var result strings.Builder

	// Parse similar rows
	result.WriteString("Relevant Data from Database Tables:\n")
	for tableName, rows := range relevantData.SimilarRows {
		result.WriteString(fmt.Sprintf("Table: %s\n", tableName))
		for i, row := range rows {
			result.WriteString(fmt.Sprintf("  Row %d:\n", i+1))
			for key, value := range row {
				result.WriteString(fmt.Sprintf("    %s: %v\n", key, value))
			}
		}
		result.WriteString("\n")
	}

	// Parse similar documents
	result.WriteString("Relevant Documents:\n")
	for i, doc := range relevantData.SimilarDocuments {
		result.WriteString(fmt.Sprintf("Document %d:\n", i+1))
		result.WriteString(fmt.Sprintf("  Collection Slug: %s\n", doc.CollectionSlug))
		result.WriteString(fmt.Sprintf("  CID: %s\n", doc.CID))
		result.WriteString(fmt.Sprintf("  Content: %s\n", doc.Content))
		result.WriteString("\n")
	}

	return result.String(), nil
}

func AnalyzeYesNoResponse(response string) bool {
	// Search for "YES" or "NO" in the entire string
	hasYes := strings.Contains(response, "YES")
	hasNo := strings.Contains(response, "NO")

	// Return true if either "YES" or "NO" is found
	return hasYes || hasNo
}

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
