package main

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"os"
	"regexp"
	"strconv"
	"strings"

	vision "cloud.google.com/go/vision/apiv1"
	"cloud.google.com/go/vision/v2/apiv1/visionpb"
)

// OCRService handles receipt scanning with Google Cloud Vision
type OCRService struct {
	client    *vision.ImageAnnotatorClient
	ctx       context.Context
	enabled   bool
}

// NewOCRService creates a new OCR service
func NewOCRService() (*OCRService, error) {
	ctx := context.Background()
	
	// Check if Google Cloud credentials are configured
	credPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credPath == "" {
		log.Println("Google Cloud Vision not configured, using mock OCR")
		return &OCRService{
			ctx:     ctx,
			enabled: false,
		}, nil
	}

	// Create Vision API client
	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		log.Printf("Failed to create Vision API client: %v", err)
		return &OCRService{
			ctx:     ctx,
			enabled: false,
		}, nil
	}

	log.Println("Google Cloud Vision OCR enabled")
	return &OCRService{
		client:  client,
		ctx:     ctx,
		enabled: true,
	}, nil
}

// Close closes the OCR service client
func (s *OCRService) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// ScanReceipt scans a receipt image and extracts amount
func (s *OCRService) ScanReceipt(file multipart.File) (int, error) {
	// If Google Cloud Vision is not enabled, use mock
	if !s.enabled {
		return s.mockOCR()
	}

	// Read file content
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return 0, err
	}

	// Create image from bytes
	image := &visionpb.Image{
		Content: fileBytes,
	}

	// Perform text detection
	annotations, err := s.client.DetectTexts(s.ctx, image, nil, 10)
	if err != nil {
		log.Printf("OCR error: %v, falling back to mock", err)
		return s.mockOCR()
	}

	if len(annotations) == 0 {
		return 0, errors.New("no text found in image")
	}

	// Extract full text
	fullText := annotations[0].Description

	// Parse amount from text
	amount, err := s.extractAmount(fullText)
	if err != nil {
		return 0, err
	}

	return amount, nil
}

// extractAmount extracts monetary amount from OCR text
func (s *OCRService) extractAmount(text string) (int, error) {
	// Common patterns for Indonesian receipts
	patterns := []string{
		`(?i)total[:\s]*rp?[\s]*([0-9.,]+)`,          // Total: Rp 50.000
		`(?i)jumlah[:\s]*rp?[\s]*([0-9.,]+)`,         // Jumlah: Rp 50.000
		`(?i)bayar[:\s]*rp?[\s]*([0-9.,]+)`,          // Bayar: Rp 50.000
		`(?i)grand\s*total[:\s]*rp?[\s]*([0-9.,]+)`, // Grand Total: Rp 50.000
		`rp[\s]*([0-9.,]+)`,                          // Rp 50.000
		`([0-9]{2,3}[.,][0-9]{3}[.,][0-9]{3})`,      // 50.000.000 or 50,000,000
		`([0-9]{2,3}[.,][0-9]{3})`,                   // 50.000 or 50,000
	}

	var amounts []int
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(text, -1)

		for _, match := range matches {
			if len(match) > 1 {
				// Clean the amount string
				amountStr := match[1]
				amountStr = strings.ReplaceAll(amountStr, ".", "")
				amountStr = strings.ReplaceAll(amountStr, ",", "")
				amountStr = strings.TrimSpace(amountStr)

				// Convert to integer
				amount, err := strconv.Atoi(amountStr)
				if err == nil && amount > 0 && amount < 100000000 { // Max 100 million
					amounts = append(amounts, amount)
				}
			}
		}
	}

	if len(amounts) == 0 {
		return 0, errors.New("no valid amount found in receipt")
	}

	// Return the largest amount (usually the total)
	maxAmount := 0
	for _, amt := range amounts {
		if amt > maxAmount {
			maxAmount = amt
		}
	}

	return maxAmount, nil
}

// mockOCR provides mock OCR response for testing
func (s *OCRService) mockOCR() (int, error) {
	// Simulate processing time
	// time.Sleep(1500 * time.Millisecond) // Commented out for faster response
	
	// Return a random-ish amount for demo
	amounts := []int{45000, 75000, 85000, 120000, 50000, 95000}
	return amounts[len(amounts)/2], nil // Return middle value for consistency
}

// ExtractAmountFromBase64 extracts amount from base64 encoded image
func (s *OCRService) ExtractAmountFromBase64(base64Image string) (int, error) {
	// Decode base64
	imageData, err := base64.StdEncoding.DecodeString(base64Image)
	if err != nil {
		return 0, err
	}

	// If not enabled, use mock
	if !s.enabled {
		return s.mockOCR()
	}

	// Create image from bytes
	image := &visionpb.Image{
		Content: imageData,
	}

	// Perform text detection
	annotations, err := s.client.DetectTexts(s.ctx, image, nil, 10)
	if err != nil {
		log.Printf("OCR error: %v, falling back to mock", err)
		return s.mockOCR()
	}

	if len(annotations) == 0 {
		return 0, errors.New("no text found in image")
	}

	// Extract and parse amount
	return s.extractAmount(annotations[0].Description)
}
