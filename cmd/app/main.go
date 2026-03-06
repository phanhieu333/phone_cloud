package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	BaseURL      = "https://api.morelogin.com"
	AppID        = "1690866404614480"
	APIKey       = "289df9a3df534fffbc249e35944ba9af"
	CloudPhoneID = 1690867972145140

	tokenRefreshBefore = 60 * time.Second // refresh trước khi hết hạn 60s
)

var (
	cachedToken  string
	cachedExpiry time.Time
)

// ─── STEP 1: Lấy Access Token ───────────────────────────────────────────────
// getAccessToken gọi API lấy token mới, trả về (token, expiresIn giây, error).
func getAccessToken() (string, int, error) {
	payload := map[string]string{
		"client_id":     AppID,
		"client_secret": APIKey,
		"grant_type":    "client_credentials",
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(
		BaseURL+"/oauth2/token",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return "", 0, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			AccessToken string `json:"access_token"`
			ExpiresIn   int    `json:"expires_in"`
		} `json:"data"`
	}

	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", 0, fmt.Errorf("parse token response failed: %w, body: %s", err, respBody)
	}
	if result.Code != 0 {
		return "", 0, fmt.Errorf("token error code %d: %s", result.Code, result.Msg)
	}

	return result.Data.AccessToken, result.Data.ExpiresIn, nil
}

// getValidAccessToken trả về token còn hiệu lực: dùng cache nếu chưa expired, không thì lấy mới.
func getValidAccessToken() (string, error) {
	if cachedToken != "" && time.Now().Before(cachedExpiry) {
		log.Println("Using cached access token (not expired).")
		return cachedToken, nil
	}
	log.Println("Token expired or missing, fetching new token...")
	token, expiresIn, err := getAccessToken()
	log.Println("Token:", token)
	if err != nil {
		return "", err
	}
	cachedToken = token
	// Hết hạn sớm hơn expiresIn một chút để tránh dùng token sắp hết hạn
	cachedExpiry = time.Now().Add(time.Duration(expiresIn)*time.Second - tokenRefreshBefore)
	log.Println("Token cached, expires at", cachedExpiry.Format(time.RFC3339))
	return token, nil
}

// ─── STEP 2: Gọi exeCommand qua Open API ────────────────────────────────────
func exeCommand(token string, cloudPhoneID int64, command string) (string, error) {
	payload := map[string]interface{}{
		"id":      cloudPhoneID,
		"command": command,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", BaseURL+"/cloudphone/exeCommand", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("exeCommand request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data string `json:"data"`
	}
	json.Unmarshal(respBody, &result)

	if result.Code != 0 {
		return "", fmt.Errorf("exeCommand error code %d: %s, body: %s", result.Code, result.Msg, respBody)
	}
	return result.Data, nil
}

// ─── STEP 3: Upload get_js.sh lên cloud phone ────────────────────────────────
func uploadFile(token string, cloudPhoneID int64, localFilePath string, uploadDest string) (string, error) {
	f, err := os.Open(localFilePath)
	if err != nil {
		return "", fmt.Errorf("open upload file failed: %w", err)
	}
	defer f.Close()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	// Doc: file (File), id (int64), uploadDest (string) — thứ tự giống doc
	part, err := w.CreateFormFile("file", filepath.Base(localFilePath))
	if err != nil {
		_ = w.Close()
		return "", fmt.Errorf("create multipart file part failed: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		_ = w.Close()
		return "", fmt.Errorf("write multipart file part failed: %w", err)
	}
	_ = w.WriteField("id", fmt.Sprintf("%d", cloudPhoneID))
	_ = w.WriteField("uploadDest", uploadDest)

	if err := w.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer failed: %w", err)
	}

	req, err := http.NewRequest("POST", BaseURL+"/cloudphone/uploadFile", &body)
	if err != nil {
		return "", fmt.Errorf("build uploadFile request failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", w.FormDataContentType())

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("uploadFile request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	_ = json.Unmarshal(respBody, &result)

	if result.Code != 0 {
		return "", fmt.Errorf("uploadFile error code %d: %s, body: %s", result.Code, result.Msg, respBody)
	}
	if len(result.Data) == 0 {
		return "", nil
	}
	return strings.TrimSpace(string(result.Data)), nil
}

func writeGetJSScript(localPath string) error {
	// Script tối giản: ưu tiên đọc giá trị từ /sdcard/jsvar.txt (nếu có),
	// rồi in ra một dòng "JSVAR=...".
	content := strings.Join([]string{
		"#!/system/bin/sh",
		"set -e",
		"",
		"JSVAR=\"\"",
		"if [ -f /sdcard/jsvar.txt ]; then",
		"  JSVAR=\"$(cat /sdcard/jsvar.txt | tr -d '\\r\\n')\"",
		"fi",
		"",
		"echo \"JSVAR=${JSVAR}\"",
		"",
	}, "\n")

	if err := os.WriteFile(localPath, []byte(content), 0o755); err != nil {
		return fmt.Errorf("write %s failed: %w", localPath, err)
	}
	return nil
}

var jsVarPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?m)^\s*JSVAR\s*=\s*(.*)\s*$`),
	regexp.MustCompile(`(?m)^\s*JSVAR\s*:\s*(.*)\s*$`),
}

func parseJSVAR(output string) (string, error) {
	for _, re := range jsVarPatterns {
		m := re.FindStringSubmatch(output)
		if len(m) == 2 {
			return strings.TrimSpace(m[1]), nil
		}
	}
	return "", fmt.Errorf("JSVAR not found in output: %s", strings.TrimSpace(output))
}

// ─── MAIN ────────────────────────────────────────────────────────────────────

func main() {
	// 1. Lấy access token (dùng cache nếu chưa expired)
	log.Println("Getting access token...")
	token, err := getValidAccessToken()
	if err != nil {
		log.Fatal("Auth failed:", err)
	}

	// 2. Mở Chrome + URL qua exeCommand
	targetURL := "https://savvysc.com/p/apparecchi-invisibili-convenienti-guida-completa-per-un-sorriso-allineato-a-costi-contenuti-524280.webm?__bt=true&_bot=2&_tag=581456_129_40_2353567_0&_type=index_link&ad_id=%7B%7Bad.id%7D%7D&ad_name=%7B%7Bad.name%7D%7D&arb_ad_id=2353567&arb_ad_id=2353567&arb_campaign_id=581456&arb_creative_id=2353567&arb_direct=on&campaign_id=%7B%7Bcampaign.id%7D%7D&campaign_name=%7B%7Bcampaign.name%7D%7D&dontRedirect=true&network=facebook&section_id=%7B%7Badset.id%7D%7D&section_name=%7B%7Badset.name%7D%7D&short_name=fbk&utm_campaign=arb-581456&utm_source=fb"
	log.Println("Opening URL via exeCommand API...")
	cmd := fmt.Sprintf(
		"am start -a android.intent.action.VIEW -n com.android.chrome/com.google.android.apps.chrome.Main -d '%s'",
		targetURL,
	)
	_, err = exeCommand(token, CloudPhoneID, cmd)
	if err != nil {
		log.Fatal("Open URL failed:", err)
	}

	// 3. Chờ Chrome khởi động
	log.Println("Waiting for Chrome to start...")
	time.Sleep(4 * time.Second)

	// 4. Tạo file get_js.sh trên PC và upload lên /sdcard/
	localScript := "get_js.sh"
	remoteScript := "/sdcard/get_js.sh"

	log.Println("Creating get_js.sh on PC...")
	if err := writeGetJSScript(localScript); err != nil {
		log.Fatal("Create get_js.sh failed:", err)
	}

	log.Println("Uploading get_js.sh to /sdcard/ via uploadFile API...")
	if _, err := uploadFile(token, CloudPhoneID, localScript, "/sdcard"); err != nil {
		log.Fatal("Upload get_js.sh failed:", err)
	}

	// 5. chmod + chạy script qua exeCommand
	log.Println("Executing get_js.sh on cloud phone...")
	output, err := exeCommand(token, CloudPhoneID, fmt.Sprintf("chmod 755 %s && sh %s", remoteScript, remoteScript))
	if err != nil {
		log.Fatal("Run get_js.sh failed:", err)
	}

	// 6. Parse JSVAR từ output
	jsvar, err := parseJSVAR(output)
	if err != nil {
		log.Fatal("Parse JSVAR failed:", err)
	}

	fmt.Println("\n=== Output ===")
	fmt.Println(output)
	fmt.Println("\n=== JSVAR ===")
	fmt.Println(jsvar)
}
