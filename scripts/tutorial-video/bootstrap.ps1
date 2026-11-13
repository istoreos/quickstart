$ErrorActionPreference='Stop'
$root=$env:TUTORIAL_SERVER_ROOT
if(-not $root) { throw 'TUTORIAL_SERVER_ROOT is required' }
New-Item -ItemType Directory -Force "$root\tool","$root\output" | Out-Null
Set-Location "$root\tool"
if(-not(Test-Path package-lock.json)) { throw 'Sync the versioned package and lock files first' }
npm.cmd ci
if($LASTEXITCODE -ne 0) { throw 'npm install failed' }
npx.cmd playwright install chromium
if($LASTEXITCODE -ne 0) { throw 'Chromium install failed' }
Write-Output 'VIDEO_TOOLS_READY'
