package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"unicode"

	"github.com/gabriel-vasile/mimetype"
	"github.com/ledongthuc/pdf"
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
		return ".pdf", nil
	case strings.HasPrefix(mimeString, "text/plain"):
		// Check if it looks like Markdown
		if looksLikeMarkdown(fileBytes) {
			return ".md", nil
		}
		return ".txt", nil
	case strings.HasPrefix(mimeString, "application/vnd.openxmlformats-officedocument.wordprocessingml.document"):
		return ".docx", nil
	case strings.HasPrefix(mimeString, "application/msword"):
		return ".doc", nil
	case strings.HasPrefix(mimeString, "text/markdown"):
		return ".md", nil
	default:
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

func ExtractTextFromPDF(fileBytes []byte) (string, error) {
	pdfReader, err := pdf.NewReader(bytes.NewReader(fileBytes), int64(len(fileBytes)))
	if err != nil {
		return "", fmt.Errorf("error opening PDF: %v", err)
	}

	var text strings.Builder
	for pageNum := 1; pageNum <= pdfReader.NumPage(); pageNum++ {
		page := pdfReader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		pageText, err := page.GetPlainText(nil)
		if err != nil {
			// Log the error but continue with other pages
			log.Printf("Error extracting text from page %d: %v", pageNum, err)
			continue
		}
		text.WriteString(pageText)
	}

	extractedText := text.String()
	cleanedText := removeNonPrintableCharacters(IdentifyAndReplaceCommonProblematicCharacters(extractedText))
	normalizedText := normalizeWhitespace(cleanedText)

	return normalizedText, nil
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
	// Replace multiple spaces with a single space
	spaceNormalized := regexp.MustCompile(`\s+`).ReplaceAllString(input, " ")
	// Ensure single newline between paragraphs
	return regexp.MustCompile(`\n\s*\n`).ReplaceAllString(spaceNormalized, "\n\n")
}

func removeNonPrintableCharacters(input string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, input)
}

func IdentifyAndReplaceCommonProblematicCharacters(input string) string {
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
