package main

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-shiori/go-readability"
	"github.com/ledongthuc/pdf"
	"github.com/xuri/excelize/v2"
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

	case ".pptx":
		return extractPPTX(path)

	case ".xlsx":
		return extractXLSX(path)

	case ".csv":
		return extractCSV(path)

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

func extractPPTX(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("open pptx: %w", err)
	}
	defer r.Close()

	var slideFiles []*zip.File
	var noteFiles []*zip.File
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			slideFiles = append(slideFiles, f)
		}
		if strings.HasPrefix(f.Name, "ppt/notesSlides/notesSlide") && strings.HasSuffix(f.Name, ".xml") {
			noteFiles = append(noteFiles, f)
		}
	}
	sort.Slice(slideFiles, func(i, j int) bool { return slideFiles[i].Name < slideFiles[j].Name })
	sort.Slice(noteFiles, func(i, j int) bool { return noteFiles[i].Name < noteFiles[j].Name })

	var sb strings.Builder
	for i, f := range slideFiles {
		rc, err := f.Open()
		if err != nil {
			continue
		}
		text := extractXMLText(rc, "t")
		rc.Close()
		sb.WriteString(fmt.Sprintf("## Slide %d\n\n%s\n\n", i+1, text))

		if i < len(noteFiles) {
			rc2, err := noteFiles[i].Open()
			if err == nil {
				notes := extractXMLText(rc2, "t")
				rc2.Close()
				if notes != "" {
					sb.WriteString(fmt.Sprintf("[Notes: %s]\n\n", notes))
				}
			}
		}
	}
	return strings.TrimSpace(sb.String()), nil
}

func extractXMLText(r io.Reader, localName string) string {
	var sb strings.Builder
	dec := xml.NewDecoder(r)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == localName {
			var text string
			if err := dec.DecodeElement(&text, &se); err == nil && text != "" {
				sb.WriteString(text)
				sb.WriteRune(' ')
			}
		}
	}
	return strings.TrimSpace(sb.String())
}

func extractCSV(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open csv: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return "", fmt.Errorf("parse csv: %w", err)
	}
	if len(records) == 0 {
		return "", nil
	}

	headers := records[0]
	rows := records[1:]

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CSV: %d rows × %d columns\n\n", len(rows), len(headers)))
	sb.WriteString("Columns: " + strings.Join(headers, ", ") + "\n\n")

	limit := 5
	if len(rows) < limit {
		limit = len(rows)
	}
	sb.WriteString(fmt.Sprintf("Sample (%d of %d rows):\n", limit, len(rows)))
	for _, row := range rows[:limit] {
		sb.WriteString("  " + strings.Join(row, " | ") + "\n")
	}

	return sb.String(), nil
}

func extractXLSX(path string) (string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return "", fmt.Errorf("open xlsx: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("XLSX: %d sheet(s)\n\n", len(sheets)))

	for _, sheet := range sheets {
		rows, err := f.GetRows(sheet)
		if err != nil || len(rows) == 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("## Sheet: %s (%d rows)\n\n", sheet, len(rows)-1))
		if len(rows) > 0 {
			sb.WriteString("Columns: " + strings.Join(rows[0], ", ") + "\n\n")
		}
		limit := 5
		if len(rows)-1 < limit {
			limit = len(rows) - 1
		}
		if limit > 0 {
			sb.WriteString(fmt.Sprintf("Sample (%d rows):\n", limit))
			for _, row := range rows[1 : 1+limit] {
				sb.WriteString("  " + strings.Join(row, " | ") + "\n")
			}
		}
		sb.WriteRune('\n')
	}
	return strings.TrimSpace(sb.String()), nil
}
