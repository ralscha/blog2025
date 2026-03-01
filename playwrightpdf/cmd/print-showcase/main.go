package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type section struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type breaksData struct {
	DocumentTitle string    `json:"documentTitle"`
	Sections      []section `json:"sections"`
}

type tableRow struct {
	SKU         string `json:"sku"`
	Description string `json:"description"`
	Qty         int    `json:"qty"`
	Unit        string `json:"unit"`
	Amount      string `json:"amount"`
}

type tableData struct {
	Title string     `json:"title"`
	Rows  []tableRow `json:"rows"`
}

type card struct {
	Heading string `json:"heading"`
	Text    string `json:"text"`
}

type pageBoxData struct {
	Title string `json:"title"`
	Cards []card `json:"cards"`
}

type showcaseExample struct {
	Name         string
	TemplatePath string
	DataPath     string
	OutFileName  string
	Loader       func([]byte) (any, error)
}

func main() {
	serviceURL := flag.String("service-url", "http://localhost:3000/pdf", "Playwright PDF service URL")
	outDir := flag.String("out-dir", ".", "Output directory for showcase PDFs")
	flag.Parse()

	examples := []showcaseExample{
		{
			Name:         "breaks",
			TemplatePath: "./templates/print-breaks.html.tmpl",
			DataPath:     "./data/print-breaks.json",
			OutFileName:  "print-showcase-breaks.pdf",
			Loader: func(raw []byte) (any, error) {
				var payload breaksData
				err := json.Unmarshal(raw, &payload)
				return payload, err
			},
		},
		{
			Name:         "table",
			TemplatePath: "./templates/print-table.html.tmpl",
			DataPath:     "./data/print-table.json",
			OutFileName:  "print-showcase-table.pdf",
			Loader: func(raw []byte) (any, error) {
				var payload tableData
				err := json.Unmarshal(raw, &payload)
				return payload, err
			},
		},
		{
			Name:         "pagebox",
			TemplatePath: "./templates/print-pagebox.html.tmpl",
			DataPath:     "./data/print-pagebox.json",
			OutFileName:  "print-showcase-pagebox.pdf",
			Loader: func(raw []byte) (any, error) {
				var payload pageBoxData
				err := json.Unmarshal(raw, &payload)
				return payload, err
			},
		},
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	for _, example := range examples {
		if err := runExample(example, *serviceURL, *outDir); err != nil {
			fmt.Fprintf(os.Stderr, "showcase %s failed: %v\n", example.Name, err)
			os.Exit(1)
		}
	}

	fmt.Printf("print showcase PDFs generated in: %s\n", *outDir)
}

func runExample(example showcaseExample, serviceURL string, outDir string) error {
	raw, err := os.ReadFile(example.DataPath)
	if err != nil {
		return fmt.Errorf("read data file %s: %w", example.DataPath, err)
	}

	payload, err := example.Loader(raw)
	if err != nil {
		return fmt.Errorf("decode data file %s: %w", example.DataPath, err)
	}

	templateRaw, err := os.ReadFile(example.TemplatePath)
	if err != nil {
		return fmt.Errorf("read template file %s: %w", example.TemplatePath, err)
	}

	htmlDoc, err := renderTemplate(example.Name, string(templateRaw), payload)
	if err != nil {
		return fmt.Errorf("render html: %w", err)
	}

	pdfBytes, err := generatePDF(serviceURL, htmlDoc)
	if err != nil {
		return fmt.Errorf("generate PDF: %w", err)
	}

	outPath := filepath.Join(outDir, example.OutFileName)
	if err := os.WriteFile(outPath, pdfBytes, 0o644); err != nil {
		return fmt.Errorf("write file %s: %w", outPath, err)
	}

	fmt.Printf("generated %s (%d bytes)\n", outPath, len(pdfBytes))
	return nil
}
func renderTemplate(name string, templateSrc string, payload any) (string, error) {
	tmpl, err := template.New(name).Funcs(template.FuncMap{
		"repeat": strings.Repeat,
	}).Parse(templateSrc)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, payload); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func generatePDF(serviceURL string, htmlDoc string) ([]byte, error) {
	client := &http.Client{Timeout: 45 * time.Second}
	req, err := http.NewRequest(http.MethodPost, serviceURL, strings.NewReader(htmlDoc))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/html; charset=utf-8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("service returned %s: %s", resp.Status, string(message))
	}

	pdfData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return pdfData, nil
}
