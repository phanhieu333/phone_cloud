#!/system/bin/sh
set -e

JSVAR=""
if [ -f /sdcard/jsvar.txt ]; then
  JSVAR="$(/system/bin/cat /sdcard/jsvar.txt | /system/bin/tr -d '\r\n')"
fi

echo "JSVAR=${JSVAR}"
