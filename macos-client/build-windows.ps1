$ErrorActionPreference = 'Stop'

Set-Location $PSScriptRoot

if (-not (Get-Command node -ErrorAction SilentlyContinue)) {
  throw '未找到 Node.js。请先安装 Node.js 22 或更新的 LTS 版本。'
}

npm install
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

npm run app:win
exit $LASTEXITCODE
