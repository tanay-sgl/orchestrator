package fileprocessing

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/russross/blackfriday/v2"
)

func GetCIDAsBytes(cid string) ([]byte, error) {
	baseURL := "http://127.0.0.1:5001/api/v0/cat"

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %w", err)
	}
	q := u.Query()
	q.Set("arg", cid)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("POST", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return body, nil
}

func GetFileChunksFromCIDAsStrings(cid string, chunkSize int) ([]string, error) {
	//fmt.Printf("GetFileChunksFromCIDAsStrings\n")
	// Get the file bytes using the provided function
	fileBytes, err := GetCIDAsBytes(cid)
	if err != nil {
		return nil, fmt.Errorf("error retrieving file from CID: %v", err)
	}

	// Identify file type
	fileType, err := IdentifyFileType(fileBytes)
	if err != nil {
		return nil, fmt.Errorf("error identifying file type: %v", err)
	}

	var text string

	switch fileType {
	case ".pdf":
		text, err = ExtractTextFromPDF(bytes.NewReader(fileBytes))
	case ".txt", ".text":
		text = ProcessText(string(fileBytes))
	case ".doc", ".docx":
		text, err = ExtractTextFromDOCX(fileBytes)
	case ".md":
		text = ProcessText(string(blackfriday.Run(fileBytes)))
	default:
		return nil, fmt.Errorf("unsupported file type: %s", fileType)
	}

	if err != nil {
		return nil, err
	}

	return SplitStringIntoStringArray(text, chunkSize), nil
}
