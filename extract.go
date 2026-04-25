package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-shiori/go-readability"
	"github.com/ledongthuc/pdf"
)

// ExtractFile reads a file and returns plain text content.
func ExtractFile(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt", ".md", ".mdx", ".rst":
		b, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(b), nil

	case ".html", ".htm":
		b, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		article, err := readability.FromReader(bytes.NewReader(b), nil)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(article.TextContent), nil

	case ".pdf":
		return extractPDF(path)

	case ".docx":
		return extractDOCX(path)

	default:
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
}

func extractPDF(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("open pdf: %w", err)
	}
	defer f.Close()

	var sb strings.Builder
	for i := 1; i <= r.NumPage(); i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		sb.WriteString(text)
		sb.WriteRune('\n')
	}
	return strings.TrimSpace(sb.String()), nil
}

func extractDOCX(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("open docx: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		defer rc.Close()

		var sb strings.Builder
		dec := xml.NewDecoder(rc)
		for {
			tok, err := dec.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", err
			}
			if se, ok := tok.(xml.StartElement); ok {
				switch se.Name.Local {
				case "t":
					var text string
					if err := dec.DecodeElement(&text, &se); err == nil && text != "" {
						sb.WriteString(text)
					}
				case "p": // paragraph break
					sb.WriteRune('\n')
				}
			}
		}
		return strings.TrimSpace(sb.String()), nil
	}
	return "", fmt.Errorf("word/document.xml not found in %s", path)
}
