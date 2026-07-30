package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type OCRService struct {
	apiKey  string
	enabled bool
}

type OCRResult struct {
	Store      string   `json:"store"`
	Items      []string `json:"items"`
	Total      int      `json:"total"`
	Date       string   `json:"date"`
	CategoryID int      `json:"category_id"`
	RawText    string   `json:"raw_text"`
}

type ocrSpaceResponse struct {
	ParsedResults []struct {
		ParsedText        string `json:"ParsedText"`
		FileParseExitCode int    `json:"FileParseExitCode"`
		ErrorMessage      string `json:"ErrorMessage"`
	} `json:"ParsedResults"`
	OCRExitCode           int      `json:"OCRExitCode"`
	IsErroredOnProcessing bool     `json:"IsErroredOnProcessing"`
	ErrorMessage          []string `json:"ErrorMessage"`
}

func NewOCRService() (*OCRService, error) {
	apiKey := os.Getenv("OCR_SPACE_API_KEY")
	if apiKey == "" {
		log.Println("OCR_SPACE_API_KEY not set, OCR will use mock")
		return &OCRService{enabled: false}, nil
	}
	log.Println("OCR.Space enabled")
	return &OCRService{apiKey: apiKey, enabled: true}, nil
}

func (s *OCRService) Close() error {
	return nil
}

func (s *OCRService) ScanReceipt(file multipart.File, categories []Category) (*OCRResult, error) {
	if !s.enabled {
		return s.mockOCRResult(), nil
	}

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	rawText, err := s.callOCRSpace(fileBytes)
	if err != nil {
		log.Printf("OCR.Space error: %v, falling back to mock", err)
		return s.mockOCRResult(), nil
	}

	if strings.TrimSpace(rawText) == "" {
		return nil, errors.New("no text found in image")
	}

	log.Printf("=== OCR RAW TEXT ===\n%s\n===================", rawText)

	ai := NewAIService()
	if ai.IsConfigured() {
		result, err := s.parseWithAI(ai, rawText, categories)
		if err == nil {
			result.RawText = rawText
			log.Printf("=== OCR AI PARSED === store=%q items=%v total=%d date=%q category_id=%d", result.Store, result.Items, result.Total, result.Date, result.CategoryID)
			return result, nil
		}
		log.Printf("AI parse failed: %v, falling back to regex", err)
	}

	result := s.parseReceiptText(rawText)
	result.RawText = rawText
	log.Printf("=== OCR REGEX PARSED === store=%q items=%v total=%d date=%q", result.Store, result.Items, result.Total, result.Date)
	return result, nil
}

func (s *OCRService) parseWithAI(ai *AIService, rawText string, categories []Category) (*OCRResult, error) {
	// Build daftar kategori untuk prompt
	var catLines []string
	for _, c := range categories {
		catLines = append(catLines, fmt.Sprintf("  - id=%d name=%q type=%q", c.ID, c.Name, c.Type))
	}
	catList := strings.Join(catLines, "\n")

	systemPrompt := `Kamu adalah parser struk belanja. Ekstrak data dari teks OCR struk dan kembalikan HANYA JSON valid tanpa penjelasan apapun.

Format JSON yang harus dikembalikan:
{
  "store": "nama toko/merchant",
  "items": ["nama item 1", "nama item 2"],
  "total": 85000,
  "date": "DD/MM/YYYY atau kosong jika tidak ada",
  "category_id": 1
}

Aturan:
- "total" adalah nilai akhir yang harus dibayar (grand total/total bayar), bukan subtotal. Integer tanpa titik/koma.
- "items" hanya nama produk/barang yang dibeli, bukan pajak/diskon/ongkir.
- "store" adalah nama toko/restoran/merchant.
- "category_id" pilih dari daftar kategori yang paling cocok berdasarkan isi struk. Gunakan id yang tepat.
- Jika data tidak ditemukan, gunakan string kosong atau array kosong.

Daftar kategori tersedia:
` + catList

	response, err := ai.Chat(systemPrompt, "Teks OCR struk:\n\n"+rawText)
	if err != nil {
		return nil, err
	}

	cleaned := strings.TrimSpace(response)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var parsed struct {
		Store      string      `json:"store"`
		Items      []string    `json:"items"`
		Total      interface{} `json:"total"`
		Date       string      `json:"date"`
		CategoryID interface{} `json:"category_id"`
	}

	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse AI response as JSON: %w — response: %s", err, cleaned)
	}

	result := &OCRResult{
		Store: parsed.Store,
		Items: parsed.Items,
		Date:  parsed.Date,
	}

	switch v := parsed.Total.(type) {
	case float64:
		result.Total = int(v)
	case string:
		clean := strings.ReplaceAll(v, ".", "")
		clean = strings.ReplaceAll(clean, ",", "")
		result.Total, _ = strconv.Atoi(strings.TrimSpace(clean))
	}

	switch v := parsed.CategoryID.(type) {
	case float64:
		result.CategoryID = int(v)
	case string:
		result.CategoryID, _ = strconv.Atoi(strings.TrimSpace(v))
	}

	return result, nil
}

func (s *OCRService) callOCRSpace(fileBytes []byte) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "receipt.jpg")
	if err != nil {
		return "", err
	}
	if _, err := part.Write(fileBytes); err != nil {
		return "", err
	}

	writer.WriteField("language", "eng")
	writer.WriteField("isOverlayRequired", "true")
	writer.WriteField("scale", "true")
	writer.WriteField("isTable", "true")
	writer.WriteField("OCREngine", "2")
	writer.WriteField("detectOrientation", "true")

	if err := writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.ocr.space/Parse/Image", body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("apikey", s.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var ocrResp ocrSpaceResponse
	if err := json.Unmarshal(respBytes, &ocrResp); err != nil {
		return "", fmt.Errorf("failed to parse OCR response: %w", err)
	}

	if ocrResp.IsErroredOnProcessing {
		return "", fmt.Errorf("OCR processing error: %v", ocrResp.ErrorMessage)
	}

	if len(ocrResp.ParsedResults) == 0 {
		return "", errors.New("no parsed results from OCR")
	}

	return ocrResp.ParsedResults[0].ParsedText, nil
}

func (s *OCRService) parseReceiptText(text string) *OCRResult {
	lines := strings.Split(text, "\n")
	var cleanLines []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			cleanLines = append(cleanLines, l)
		}
	}

	result := &OCRResult{}

	nonDigitRe := regexp.MustCompile(`^[0-9\s\.\,\:\-\/\(\)]+$`)
	for _, line := range cleanLines[:minInt(6, len(cleanLines))] {
		if len(line) >= 3 && !nonDigitRe.MatchString(line) {
			result.Store = line
			break
		}
	}

	datePatterns := []string{
		`(\d{1,2}[\/\-]\d{1,2}[\/\-]\d{2,4})`,
		`(\d{1,2}\s+(?:Jan|Feb|Mar|Apr|Mei|Jun|Jul|Agu|Sep|Okt|Nov|Des)\w*\s+\d{2,4})`,
		`(\d{4}[\/\-]\d{2}[\/\-]\d{2})`,
	}
	for _, pat := range datePatterns {
		re := regexp.MustCompile(`(?i)` + pat)
		if m := re.FindStringSubmatch(text); len(m) > 1 {
			result.Date = m[1]
			break
		}
	}

	skipRe := regexp.MustCompile(`(?i)total|subtotal|grand|bayar|kembali|tunai|cash|ppn|tax|diskon|discount|kembalian|point|member|kasir|terima|thank|void|cancel|struk|nota|invoice|receipt|alamat|address|telp|phone|npwp`)

	itemWithPriceRe := regexp.MustCompile(`^(.{3,35}?)\s{2,}[\d\.\,]+\s*$`)
	for _, line := range cleanLines {
		if skipRe.MatchString(line) {
			continue
		}
		if m := itemWithPriceRe.FindStringSubmatch(line); len(m) > 1 {
			itemName := strings.TrimSpace(m[1])
			if len(itemName) >= 3 {
				result.Items = append(result.Items, itemName)
			}
		}
	}

	result.Total, _ = s.extractAmount(text)
	return result
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func detectMimeType(data []byte) string {
	if len(data) >= 4 {
		if data[0] == 0xFF && data[1] == 0xD8 {
			return "image/jpeg"
		}
		if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
			return "image/png"
		}
		if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 {
			return "image/gif"
		}
		if data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 {
			return "image/webp"
		}
	}
	return "image/jpeg"
}

func (s *OCRService) extractAmount(text string) (int, error) {
	patterns := []string{
		`(?i)grand\s*total[:\s]*rp?\.?\s*([0-9.,]+)`,
		`(?i)total\s*bayar[:\s]*rp?\.?\s*([0-9.,]+)`,
		`(?i)total[:\s]*rp?\.?\s*([0-9.,]+)`,
		`(?i)jumlah[:\s]*rp?\.?\s*([0-9.,]+)`,
		`(?i)bayar[:\s]*rp?\.?\s*([0-9.,]+)`,
		`(?i)tagihan[:\s]*rp?\.?\s*([0-9.,]+)`,
		`rp\.?\s*([0-9.,]+)`,
		`([0-9]{1,3}(?:\.[0-9]{3})+)`,
	}

	var amounts []int
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) > 1 {
				amountStr := strings.ReplaceAll(match[1], ".", "")
				amountStr = strings.ReplaceAll(amountStr, ",", "")
				amountStr = strings.TrimSpace(amountStr)
				if amount, err := strconv.Atoi(amountStr); err == nil && amount > 100 && amount < 100000000 {
					amounts = append(amounts, amount)
				}
			}
		}
	}

	if len(amounts) == 0 {
		return 0, errors.New("no valid amount found in receipt")
	}

	maxAmount := 0
	for _, amt := range amounts {
		if amt > maxAmount {
			maxAmount = amt
		}
	}
	return maxAmount, nil
}

func (s *OCRService) mockOCRResult() *OCRResult {
	return &OCRResult{
		Store:   "Alfamart",
		Items:   []string{"Indomie Goreng", "Teh Botol Sosro", "Chitato"},
		Total:   85000,
		Date:    "",
		RawText: "",
	}
}
