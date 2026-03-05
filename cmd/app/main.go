package main

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	adbDevice      = "98.98.125.30:20839"
	connectionCode = "PDpUXPWV" // Mã kết nối (từ tooltip), có thể ghi đè bằng env ADB_PAIR_CODE
	maxConnect     = 5
	connectWait    = 2 * time.Second
)

func runADB(args ...string) ([]byte, error) {
	full := append([]string{"adb"}, args...)
	log.Println("[exec]", strings.Join(full, " "))
	return exec.Command("adb", args...).CombinedOutput()
}

func runADBWithDevice(args ...string) ([]byte, error) {
	a := append([]string{"-s", adbDevice}, args...)
	full := append([]string{"adb"}, a...)
	log.Println("[exec]", strings.Join(full, " "))
	return exec.Command("adb", a...).CombinedOutput()
}

// Khởi động lại ADB server để tránh connection cũ bị "closed"
func ensureADBServer() {
	log.Println("Restarting ADB server...")
	runADB("kill-server")
	time.Sleep(500 * time.Millisecond)
	if out, err := runADB("start-server"); err != nil {
		log.Fatalf("adb start-server failed: %v\n%s", err, out)
	}
	time.Sleep(500 * time.Millisecond)
}

// Pair với thiết bị qua mã kết nối (Wireless debugging Android 11+)
func pairDevice() {
	code := os.Getenv("ADB_PAIR_CODE")
	if code == "" {
		code = connectionCode
	}
	log.Println("Pairing with device (code:", code, ")...")
	log.Println("[exec] adb pair", adbDevice, "< (code)")
	cmd := exec.Command("adb", "pair", adbDevice)
	cmd.Stdin = bytes.NewBufferString(code + "\n")
	out, err := cmd.CombinedOutput()
	log.Println(strings.TrimSpace(string(out)))
	if err != nil {
		// Pair có thể bỏ qua nếu đã pair rồi hoặc thiết bị không dùng pair
		log.Printf("Pair warning (continuing): %v", err)
	}
	time.Sleep(1 * time.Second)
}

// Kết nối tới thiết bị, retry vài lần
func connectDevice() {
	ensureADBServer()
	pairDevice()
	for i := 0; i < maxConnect; i++ {
		log.Printf("Connecting to %s (attempt %d/%d)...", adbDevice, i+1, maxConnect)
		out, err := runADB("connect", adbDevice)
		log.Println(strings.TrimSpace(string(out)))
		if err != nil {
			log.Printf("connect error: %v", err)
			time.Sleep(connectWait)
			continue
		}
		if strings.Contains(string(out), "connected") || strings.Contains(string(out), "already connected") {
			time.Sleep(connectWait) // đợi device ổn định
			if deviceReady() {
				log.Println("Device ready.")
				return
			}
		}
		time.Sleep(connectWait)
	}
	log.Fatalf("Could not connect to %s after %d attempts", adbDevice, maxConnect)
}

// Kiểm tra thiết bị ở trạng thái "device" (không offline/unauthorized)
func deviceReady() bool {
	out, err := runADB("devices")
	if err != nil {
		return false
	}
	// adb devices: "98.98.125.30:20839    device"
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == adbDevice && fields[1] == "device" {
			return true
		}
	}
	return false
}

func main() {
	connectDevice()

	out, err := runADB("devices")
	if err != nil {
		log.Fatalf("adb devices failed: %v\n%s", err, out)
	}

	// Mở link trên thiết bị (mặc định hoặc từ tham số dòng lệnh)
	link := "https://tipssearch.com/p/understanding-retirement-monthly-income-a-guide-to-averages-and-age-based-variations-545886.webm?__bt=true&_bot=2&_tag=593509_130_40_2384674_0&_type=index_link&ad_id=%7B%7Bad.id%7D%7D&ad_name=%7B%7Bad.name%7D%7D&arb_ad_id=2384674&arb_ad_id=2384674&arb_campaign_id=593509&arb_creative_id=2384674&arb_direct=on&campaign_id=%7B%7Bcampaign.id%7D%7D&campaign_name=%7B%7Bcampaign.name%7D%7D&dontRedirect=true&network=facebook&section_id=%7B%7Badset.id%7D%7D&section_name=%7B%7Badset.name%7D%7D&short_name=fbk&utm_campaign=arb-593509&utm_source=fb"
	if len(os.Args) > 1 {
		link = os.Args[1]
	}

	// Quote URL cho shell trên thiết bị (tránh & ? = bị hiểu sai)
	shellCmd := "am start -a android.intent.action.VIEW -d '" + strings.ReplaceAll(link, "'", "'\\''") + "'"
	log.Println("[exec] adb shell", shellCmd)
	out, err = runADBWithDevice("shell", shellCmd)
	if err != nil {
		log.Printf("open link (VIEW) failed: %v\nOutput: %s", err, out)
		shellCmdChrome := "am start -n com.android.chrome/com.google.android.apps.chrome.Main -a android.intent.action.VIEW -d '" + strings.ReplaceAll(link, "'", "'\\''") + "'"
		out, err = runADBWithDevice("shell", shellCmdChrome)
		if err != nil {
			log.Fatalf("open link (Chrome) failed: %v\nOutput: %s", err, out)
		}
	}
	log.Println(strings.TrimSpace(string(out)))
}
