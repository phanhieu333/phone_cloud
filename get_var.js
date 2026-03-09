#!/usr/bin/env node
/**
 * Chạy trên cloud phone: kết nối Chrome qua CDP, chạy JS trong trang, ghi kết quả ra /sdcard/jsvar.txt
 * Cần: Node.js trên thiết bị, Chrome mở với remote debugging (port 9222).
 * Cài: npm install chrome-remote-interface (hoặc copy node_modules lên device)
 */

const fs = require('fs');
const outFile = '/sdcard/jsvar.txt';

// Đoạn JS chạy trong trang để lấy biến — sửa theo biến thật (vd. window.__JSVAR, window.someVar)
const EVAL_SCRIPT = `
  (function() {
    if (typeof window.__JSVAR !== 'undefined') return String(window.__JSVAR);
    if (typeof window.jsvar !== 'undefined') return String(window.jsvar);
    return '';
  })();
`;

async function main() {
  let CDP;
  try {
    CDP = require('chrome-remote-interface');
  } catch (e) {
    fs.writeFileSync(outFile, '');
    console.error('Cần cài: npm install chrome-remote-interface');
    process.exit(1);
  }

  const host = '127.0.0.1';
  const port = 9222;

  try {
    const client = await CDP({ host, port });
    const { Runtime } = client;
    await Runtime.enable();
    const { result } = await Runtime.evaluate({ expression: EVAL_SCRIPT });
    let value = '';
    if (result.type === 'string') value = result.value;
    else if (result.subtype === 'null' || result.type === 'undefined') value = '';
    else value = result.value != null ? String(result.value) : '';
    await client.close();
    fs.writeFileSync(outFile, value);
    console.log('Wrote jsvar:', value ? value.slice(0, 80) + (value.length > 80 ? '...' : '') : '(empty)');
  } catch (err) {
    console.error(err.message);
    fs.writeFileSync(outFile, '');
    process.exit(1);
  }
}

main();
