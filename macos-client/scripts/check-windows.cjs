const { execFileSync, spawn } = require('node:child_process');
const http = require('node:http');
const path = require('node:path');

const appPath = process.argv[2]
  ? path.resolve(process.argv[2])
  : path.resolve(__dirname, '..', 'release', 'win-unpacked', '163MUSIC.exe');
const testDataPath = path.resolve(__dirname, '..', '.test-user-data');
const debugPort = 9323;

function getJson(url) {
  return new Promise((resolve, reject) => {
    const request = http.get(url, (response) => {
      const chunks = [];
      response.on('data', (chunk) => chunks.push(chunk));
      response.on('end', () => {
        try {
          resolve(JSON.parse(Buffer.concat(chunks).toString('utf8')));
        } catch (error) {
          reject(error);
        }
      });
    });
    request.on('error', reject);
    request.setTimeout(1000, () => request.destroy(new Error('timeout')));
  });
}

async function waitForPage() {
  for (let attempt = 0; attempt < 40; attempt += 1) {
    try {
      const pages = await getJson(`http://127.0.0.1:${debugPort}/json`);
      const page = pages.find((item) => item.type === 'page');
      if (page?.webSocketDebuggerUrl) return page;
    } catch {}
    await new Promise((resolve) => setTimeout(resolve, 250));
  }
  throw new Error('应用窗口未在 10 秒内就绪');
}

function evaluate(webSocketUrl, expression) {
  return new Promise((resolve, reject) => {
    const socket = new WebSocket(webSocketUrl);
    const timer = setTimeout(() => reject(new Error('渲染检查超时')), 5000);
    socket.addEventListener('open', () => {
      socket.send(JSON.stringify({
        id: 1,
        method: 'Runtime.evaluate',
        params: { expression, returnByValue: true }
      }));
    });
    socket.addEventListener('message', (event) => {
      const message = JSON.parse(String(event.data));
      if (message.id !== 1) return;
      clearTimeout(timer);
      socket.close();
      if (message.error || message.result?.exceptionDetails) {
        reject(new Error(`渲染进程执行失败：${JSON.stringify(message)}`));
        return;
      }
      resolve(message.result?.result?.value);
    });
    socket.addEventListener('error', () => reject(new Error('无法连接渲染进程')));
  });
}

async function main() {
  const child = spawn(appPath, [
    `--remote-debugging-port=${debugPort}`,
    `--user-data-dir=${testDataPath}`
  ], { detached: false, stdio: 'ignore', windowsHide: true });

  try {
    const page = await waitForPage();
    await new Promise((resolve) => setTimeout(resolve, 1500));
    const serialized = await evaluate(page.webSocketDebuggerUrl, `JSON.stringify({
      title: document.title,
      text: document.body.innerText.slice(0, 240),
      rootChildren: document.querySelector('#root')?.children.length || 0
    })`);
    const result = JSON.parse(serialized || 'null');
    if (!result || result.rootChildren < 1 || !result.text.includes('163MUSIC')) {
      throw new Error(`渲染结果异常：${JSON.stringify(result)}`);
    }
    console.log(JSON.stringify(result, null, 2));
  } finally {
    if (process.platform === 'win32' && child.pid) {
      try {
        execFileSync('taskkill', ['/pid', String(child.pid), '/T', '/F'], { stdio: 'ignore' });
      } catch {}
    } else {
      child.kill('SIGTERM');
    }
  }
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
