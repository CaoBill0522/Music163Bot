const path = require('node:path');
const { _electron: electron } = require('playwright');

async function main() {
  const appPath = path.resolve(__dirname, '..', 'release', 'mac-arm64', '163MUSIC.app', 'Contents', 'MacOS', '163MUSIC');
  const app = await electron.launch({ executablePath: appPath });
  const win = await app.firstWindow({ timeout: 15000 });
  await win.waitForLoadState('domcontentloaded');
  await win.waitForTimeout(2500);
  const text = await win.locator('body').innerText({ timeout: 5000 });
  const rootHtml = await win.locator('#root').evaluate((node) => node.innerHTML.slice(0, 200));
  const screenshotPath = path.resolve(__dirname, '..', 'render-check.png');
  await win.screenshot({ path: screenshotPath });
  console.log(JSON.stringify({
    title: await win.title(),
    bodyTextLength: text.length,
    bodyTextPreview: text.slice(0, 160),
    rootHtmlLength: rootHtml.length,
    screenshotPath
  }, null, 2));
  await app.close();
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
