#!/system/bin/sh
# Dùng full path cho cat/tr — trên Android cloud phone thường không có trong PATH.
set -e

JSVAR=""
if [ -f /sdcard/jsvar.txt ]; then
  JSVAR="$(/system/bin/cat /sdcard/jsvar.txt 2>/dev/null | /system/bin/tr -d '\r\n' 2>/dev/null)" || JSVAR="$(/system/bin/cat /sdcard/jsvar.txt 2>/dev/null)"
fi

echo "JSVAR=${JSVAR}"

