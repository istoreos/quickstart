# Initialize voice-only work from verified raw-footage receipts, never captioned chapters.
$ErrorActionPreference='Stop'
if (!$env:TUTORIAL_PARENT_OUTPUT -or $env:TUTORIAL_PARENT_OUTPUT -eq $env:TUTORIAL_OUTPUT) { throw 'A separate voice revision output is required' }
if (Test-Path ($env:TUTORIAL_OUTPUT+'/parent-manifest.json')) { throw 'Revision already initialized; preserve its review history' }
$receipt=Get-Content -Raw -Encoding UTF8 ($env:TUTORIAL_PARENT_OUTPUT+'/manifest.json') | ConvertFrom-Json
$actual=(Get-FileHash ($env:TUTORIAL_PARENT_OUTPUT+'/fastnet-home-card-zh-CN.mp4') -Algorithm SHA256).Hash.ToLower()
if ($actual -ne $receipt.videoSha256) { throw 'Parent movie no longer matches its receipt' }
foreach ($dir in @('capture','render')) { New-Item -ItemType Directory -Force ($env:TUTORIAL_OUTPUT+'/'+$dir) | Out-Null }
Copy-Item ($env:TUTORIAL_PARENT_OUTPUT+'/capture/*.json') ($env:TUTORIAL_OUTPUT+'/capture')
Copy-Item ($env:TUTORIAL_PARENT_OUTPUT+'/manifest.json') ($env:TUTORIAL_OUTPUT+'/parent-manifest.json')
Copy-Item ($env:TUTORIAL_PARENT_OUTPUT+'/edit-plan.json') ($env:TUTORIAL_OUTPUT+'/parent-edit-plan.json')
Write-Output 'Voice revision initialized; no captioned video intermediates copied'
