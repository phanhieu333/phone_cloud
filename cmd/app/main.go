package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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
	tokenCacheFile     = ".morelogin_token"
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

// loadTokenFromFile đọc token + expiry từ file cache, trả về (token, expiry, true) nếu hợp lệ.
func loadTokenFromFile() (string, time.Time, bool) {
	data, err := os.ReadFile(tokenCacheFile)
	if err != nil {
		return "", time.Time{}, false
	}
	var v struct {
		Token  string `json:"token"`
		Expiry string `json:"expiry"`
	}
	if json.Unmarshal(data, &v) != nil || v.Token == "" || v.Expiry == "" {
		return "", time.Time{}, false
	}
	expiry, err := time.Parse(time.RFC3339, v.Expiry)
	if err != nil || time.Now().After(expiry) {
		return "", time.Time{}, false
	}
	return v.Token, expiry, true
}

// saveTokenToFile ghi token + expiry ra file cache.
func saveTokenToFile(token string, expiry time.Time) {
	v := struct {
		Token  string `json:"token"`
		Expiry string `json:"expiry"`
	}{Token: token, Expiry: expiry.Format(time.RFC3339)}
	data, _ := json.Marshal(v)
	if err := os.WriteFile(tokenCacheFile, data, 0o600); err != nil {
		log.Println("Warning: could not write token cache file:", err)
		return
	}
	log.Println("Token saved to cache file:", tokenCacheFile)
}

// clearTokenCache xóa token trong memory và file (dùng khi API trả 35002 để lần sau lấy token mới).
func clearTokenCache() {
	cachedToken = ""
	cachedExpiry = time.Time{}
	_ = os.Remove(tokenCacheFile)
	log.Println("Token cache cleared (will fetch new token next time).")
}

// isAuthError trả về true nếu lỗi là xác thực API (35002, authentication failed).
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "35002") || strings.Contains(s, "authentication failed")
}

// getValidAccessToken trả về token còn hiệu lực: dùng cache (memory hoặc file) nếu chưa expired, không thì lấy mới.
func getValidAccessToken() (string, error) {
	now := time.Now()
	if cachedToken != "" && now.Before(cachedExpiry) {
		log.Println("Using cached access token (memory, not expired).")
		return cachedToken, nil
	}
	if token, expiry, ok := loadTokenFromFile(); ok {
		cachedToken = token
		cachedExpiry = expiry
		log.Println("Using cached access token (from file, expires at", cachedExpiry.Format(time.RFC3339)+").")
		return cachedToken, nil
	}
	log.Println("Token expired or missing, fetching new token...")
	token, expiresIn, err := getAccessToken()
	if err != nil {
		return "", err
	}
	cachedToken = token
	cachedExpiry = now.Add(time.Duration(expiresIn)*time.Second - tokenRefreshBefore)
	saveTokenToFile(cachedToken, cachedExpiry)
	log.Println("Token cached (memory + file), expires at", cachedExpiry.Format(time.RFC3339))
	return token, nil
}

// ─── STEP 2: Gọi exeCommand qua Open API ────────────────────────────────────
func exeCommand(token string, cloudPhoneID int64, command string) (string, error) {
	log.Println("Executing command:", command)
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
	jsonData := json.Unmarshal(respBody, &result)

	log.Println("jsonData:", jsonData)

	if result.Code != 0 {
		return "", fmt.Errorf("exeCommand error code %d: %s, body: %s", result.Code, result.Msg, respBody)
	}
	return result.Data, nil
}

// ─── STEP 3: Upload get_js.sh lên cloud phone ────────────────────────────────
// Theo MoreLogin API (Cloud Phone / File Management):
// 1) Get file upload URL (POST /cloudphone/uploadUrl) → nhận presignedUrl.
// 2) Dùng HTTPS PUT upload file lên presignedUrl (upload to cloud storage).
// 3) Sau khi PUT xong mới gọi Uploading files (POST /cloudphone/uploadFile).
func uploadScriptToDevice(token string, deviceID int64, scriptContent, fileName string) error {
	// Bước 1: POST /cloudphone/uploadUrl → nhận presignedUrl
	payload, _ := json.Marshal(map[string]interface{}{
		"id":       deviceID,
		"fileName": fileName,
	})
	req, _ := http.NewRequest("POST", BaseURL+"/cloudphone/uploadUrl", bytes.NewBuffer(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("get file upload URL: %w", err)
	}
	defer resp.Body.Close()

	var urlResult struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			PresignedUrl string `json:"presignedUrl"`
		} `json:"data"`
	}
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &urlResult)
	if urlResult.Code != 0 {
		return fmt.Errorf("uploadUrl error %d: %s", urlResult.Code, urlResult.Msg)
	}
	presignedUrl := urlResult.Data.PresignedUrl
	log.Println("Step 1 OK - presignedUrl:", presignedUrl[:50]+"...")

	// Bước 2: PUT file lên S3 qua presignedUrl
	putReq, _ := http.NewRequest("PUT", presignedUrl, bytes.NewBufferString(scriptContent))
	putReq.Header.Set("Content-Type", "application/octet-stream")
	putResp, err := http.DefaultClient.Do(putReq)
	if err != nil {
		return fmt.Errorf("PUT to S3: %w", err)
	}
	io.Copy(io.Discard, putResp.Body)
	putResp.Body.Close()
	if putResp.StatusCode < 200 || putResp.StatusCode >= 300 {
		return fmt.Errorf("PUT failed: %s", putResp.Status)
	}
	log.Println("Step 2 OK - file uploaded to S3")

	// Bước 3: POST /cloudphone/uploadFile - JSON
	payload3, _ := json.Marshal(map[string]interface{}{
		"id":         deviceID,
		"url":        presignedUrl,
		"uploadDest": "/sdcard/",
	})
	req3, _ := http.NewRequest("POST", BaseURL+"/cloudphone/uploadFile", bytes.NewBuffer(payload3))
	req3.Header.Set("Authorization", "Bearer "+token)
	req3.Header.Set("Content-Type", "application/json")

	resp3, err := http.DefaultClient.Do(req3)
	if err != nil {
		return fmt.Errorf("uploadFile: %w", err)
	}
	defer resp3.Body.Close()

	respBody3, _ := io.ReadAll(resp3.Body)
	log.Println("Step 3 response:", string(respBody3))

	var result3 struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	json.Unmarshal(respBody3, &result3)
	if result3.Code != 0 {
		return fmt.Errorf("uploadFile error %d: %s", result3.Code, result3.Msg)
	}
	log.Println("Step 3 OK - file copied to device")
	return nil
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
	scriptContent, err := os.ReadFile(localScript)
	if err != nil {
		log.Fatal("Read get_js.sh failed:", err)
	}
	if err := uploadScriptToDevice(token, CloudPhoneID, string(scriptContent), "get_js.sh"); err != nil {
		log.Fatal("Upload get_js.sh failed:", err)
	}

	// 5. Chạy script qua exeCommand (không dùng chmod — API/device có thể không cho)
	log.Println("Executing get_js.sh on cloud phone (sh file)...")
	runCmd := fmt.Sprintf("sh %s", remoteScript)
	output, err := exeCommand(token, CloudPhoneID, runCmd)
	if err != nil {

		// Lỗi do lệnh (không phải auth): fallback sang cat file | sh.
		log.Println("Command failed (not auth), trying fallback: cat file | sh ...")
		runCmd = fmt.Sprintf("cat %s | sh", remoteScript)
		output, err = exeCommand(token, CloudPhoneID, runCmd)
	}
	log.Println("Output:", output)
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
