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
	"strings"
	"time"
)

type Invoice struct {
	Number      string
	Date        string
	DueDate     string
	LogoDataURI template.URL
	From        Party
	To          Party
	Items       []LineItem
	Notes       string
	Subtotal    string
	TaxLabel    string
	TaxAmount   string
	Total       string
	Currency    string
	PaymentRef  string
}

type Party struct {
	Name    string
	Address string
	Email   string
}

type LineItem struct {
	Description string
	Quantity    int
	UnitPrice   string
	Amount      string
}

func main() {
	serviceURL := flag.String("service-url", "http://localhost:3000/pdf", "Playwright PDF service URL")
	outPath := flag.String("out", "invoice.pdf", "Output PDF file path")
	dataPath := flag.String("data", "./data/invoice.json", "Path to invoice JSON data file")
	templatePath := flag.String("template", "./templates/invoice.html.tmpl", "Path to invoice HTML template")
	flag.Parse()

	invoice, err := loadInvoiceFromFile(*dataPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load invoice data: %v\n", err)
		os.Exit(1)
	}

	htmlDoc, err := renderInvoiceHTML(*templatePath, invoice)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to render invoice HTML: %v\n", err)
		os.Exit(1)
	}

	pdfBytes, err := generatePDF(*serviceURL, htmlDoc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate PDF via Playwright service: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outPath, pdfBytes, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write PDF file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("invoice PDF generated: %s (%d bytes)\n", *outPath, len(pdfBytes))
}

func loadInvoiceFromFile(path string) (Invoice, error) {
	var invoice Invoice

	raw, err := os.ReadFile(path)
	if err != nil {
		return invoice, err
	}

	if err := json.Unmarshal(raw, &invoice); err != nil {
		return invoice, err
	}

	return invoice, nil
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
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("service returned %s: %s", resp.Status, string(msg))
	}

	pdf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return pdf, nil
}

func renderInvoiceHTML(templatePath string, invoice Invoice) (string, error) {
	templateRaw, err := os.ReadFile(templatePath)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("invoice").Parse(string(templateRaw))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, invoice); err != nil {
		return "", err
	}

	return buf.String(), nil
}
