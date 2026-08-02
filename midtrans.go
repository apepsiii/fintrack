package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func midtransBaseURL(env string) string {
	if env == "production" {
		return "https://app.midtrans.com"
	}
	return "https://app.sandbox.midtrans.com"
}

func midtransAPIURL(env string) string {
	if env == "production" {
		return "https://api.midtrans.com"
	}
	return "https://api.sandbox.midtrans.com"
}

func createMidtransTransaction(serverKey, env, orderID string, amount int, name, email string) (snapToken, redirectURL string, err error) {
	url := midtransBaseURL(env) + "/snap/v1/transactions"

	payload := map[string]interface{}{
		"transaction_details": map[string]interface{}{
			"order_id":     orderID,
			"gross_amount": amount,
		},
		"customer_details": map[string]interface{}{
			"first_name": name,
			"email":      email,
		},
		"item_details": []map[string]interface{}{
			{
				"id":       "PREMIUM-1M",
				"price":    amount,
				"quantity": 1,
				"name":     "FinTrack Premium - 1 Bulan",
			},
		},
		"expiry": map[string]interface{}{
			"start_time": time.Now().Format("2006-01-02 15:04:05 +0700"),
			"unit":       "hour",
			"duration":   24,
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", "", err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(serverKey + ":"))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", "", fmt.Errorf("failed to parse Midtrans response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		errMsg, _ := result["error_messages"].([]interface{})
		if len(errMsg) > 0 {
			return "", "", fmt.Errorf("midtrans error: %v", errMsg[0])
		}
		return "", "", fmt.Errorf("midtrans error status %d: %s", resp.StatusCode, string(respBody))
	}

	snapToken, _ = result["token"].(string)
	redirectURL, _ = result["redirect_url"].(string)

	return snapToken, redirectURL, nil
}

func verifyMidtransPayment(serverKey, env, orderID string) (string, error) {
	url := fmt.Sprintf("%s/v2/%s/status", midtransAPIURL(env), orderID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(serverKey + ":"))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	status, _ := result["transaction_status"].(string)
	if status == "" {
		return "", fmt.Errorf("no transaction_status in response: %s", string(respBody))
	}

	return status, nil
}
