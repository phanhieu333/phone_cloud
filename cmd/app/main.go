package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	BaseURL      = "https://api.morelogin.com"
	AppID        = "1690866404614480"
	APIKey       = "289df9a3df534fffbc249e35944ba9af"
	DeviceADB    = "98.98.125.30:20698"
	CloudPhoneID = 1690867972145140
)

// ─── STEP 1: Lấy Access Token ───────────────────────────────────────────────
func getAccessToken() (string, error) {
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
		return "", fmt.Errorf("token request failed: %w", err)
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
		return "", fmt.Errorf("parse token response failed: %w, body: %s", err, respBody)
	}
	if result.Code != 0 {
		return "", fmt.Errorf("token error code %d: %s", result.Code, result.Msg)
	}

	return result.Data.AccessToken, nil
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

// ─── STEP 3: ADB forward port 9222 ──────────────────────────────────────────

func setupADBForward() error {
	exec.Command("adb", "connect", DeviceADB).Run()
	time.Sleep(time.Second)

	out, err := exec.Command("adb", "-s", DeviceADB,
		"forward", "tcp:9222", "localabstract:chrome_devtools_remote",
	).CombinedOutput()
	log.Printf("ADB forward output: %s", out)
	if err != nil {
		return fmt.Errorf("adb forward failed: %w, output: %s", err, out)
	}
	log.Println("ADB forward OK:", string(out))
	return nil
}

// ─── STEP 4: Lấy JS variable qua CDP ────────────────────────────────────────
func getJSVariable(expression string) (interface{}, error) {
	allocCtx, cancel := chromedp.NewRemoteAllocator(
		context.Background(),
		"http://localhost:9222",
	)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	var result interface{}
	err := chromedp.Run(ctx,
		chromedp.Evaluate(expression, &result),
	)
	if err != nil {
		return nil, fmt.Errorf("CDP evaluate error: %w", err)
	}
	return result, nil
}

// ─── MAIN ────────────────────────────────────────────────────────────────────

func main() {
	// 1. Lấy access token
	log.Println("Getting access token...")
	token, err := getAccessToken()
	if err != nil {
		log.Fatal("Auth failed:", err)
	}
	log.Println("Token OK:", token[:20]+"...")

	// 2. Setup ADB forward
	log.Println("Setting up ADB forward port 9222...")
	if err := setupADBForward(); err != nil {
		log.Fatal("ADB forward failed:", err)
	}

	// 3. Mở Chrome + URL qua exeCommand
	targetURL := "https://savvysc.com/p/apparecchi-invisibili-convenienti-guida-completa-per-un-sorriso-allineato-a-costi-contenuti-524280.webm?__bt=true&_bot=2&_tag=581456_129_40_2353567_0&_type=index_link&ad_id=%7B%7Bad.id%7D%7D&ad_name=%7B%7Bad.name%7D%7D&arb_ad_id=2353567&arb_ad_id=2353567&arb_campaign_id=581456&arb_creative_id=2353567&arb_direct=on&campaign_id=%7B%7Bcampaign.id%7D%7D&campaign_name=%7B%7Bcampaign.name%7D%7D&dontRedirect=true&network=facebook&section_id=%7B%7Badset.id%7D%7D&section_name=%7B%7Badset.name%7D%7D&short_name=fbk&utm_campaign=arb-581456&utm_source=fb"
	log.Println("Opening URL via exeCommand API...")
	cmd := fmt.Sprintf(
		"am start -a android.intent.action.VIEW -n com.android.chrome/com.google.android.apps.chrome.Main -d '%s'",
		targetURL,
	)
	result, err := exeCommand(token, CloudPhoneID, cmd)
	if err != nil {
		log.Fatal("Open URL failed:", err)
	}
	log.Println("exeCommand result:", result)

	// 4. Chờ page load
	log.Println("Waiting for page load...")
	time.Sleep(4 * time.Second)

	// 5. Lấy JS variable qua CDP
	log.Println("Fetching JS variable via CDP...")
	val, err := getJSVariable("window.relatedAdShowing")
	if err != nil {
		log.Fatal("Get JS variable failed:", err)
	}

	out, _ := json.MarshalIndent(val, "", "  ")
	fmt.Println("\n=== Result ===")
	fmt.Println(string(out))
}
