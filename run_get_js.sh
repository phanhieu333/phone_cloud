#!/system/bin/sh
# Script chạy trên cloud phone: nhận URL qua $1, mở Chrome với URL đó.
# Sau khi script này xong, app sẽ gọi exeCommand: node /sdcard/get_var.js để lấy biến JS và ghi /sdcard/jsvar.txt.

set -e

URL="${1:-}"
if [ -z "$URL" ]; then
  echo "Usage: $0 <url>"
  exit 1
fi

# 1. Mở Chrome với URL (nếu cần remote debugging cho get_var.js thì mở Chrome với port 9222, tùy image)
am start -a android.intent.action.VIEW -n com.android.chrome/com.google.android.apps.chrome.Main -d "$URL"

# 2. Chờ trang load (sau đó app sẽ chạy node /sdcard/get_var.js để lấy biến và ghi /sdcard/jsvar.txt)
sleep 10

# Nếu không dùng get_var.js (Node + CDP) mà dùng Chrome extension ghi sẵn /sdcard/jsvar.txt thì không cần gì thêm.
# Nếu dùng get_var.js: thiết bị cần Node.js và Chrome mở với remote debugging port 9222.
exit 0
