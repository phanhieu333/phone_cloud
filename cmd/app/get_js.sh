#!/system/bin/sh
set -e

JSVAR=""
if [ -f /sdcard/jsvar.txt ]; then
  JSVAR="$(cat /sdcard/jsvar.txt | tr -d '\r\n')"
fi

echo "JSVAR=${JSVAR}"
