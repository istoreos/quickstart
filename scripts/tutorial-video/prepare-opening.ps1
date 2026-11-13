# Initialize a separately authorized revision. Never clear review history or touch the parent.
$ErrorActionPreference='Stop'
if (!$env:TUTORIAL_PARENT_OUTPUT -or $env:TUTORIAL_OUTPUT -eq $env:TUTORIAL_PARENT_OUTPUT) { throw 'An isolated revision output is required' }
if (Test-Path ($env:TUTORIAL_OUTPUT+'/parent-manifest.json')) { throw 'Revision already initialized; use its existing review history' }
$receipt=Get-Content -Raw -Encoding UTF8 ($env:TUTORIAL_PARENT_OUTPUT+'/manifest.json') | ConvertFrom-Json
$actual=(Get-FileHash ($env:TUTORIAL_PARENT_OUTPUT+'/fastnet-home-card-zh-CN.mp4') -Algorithm SHA256).Hash.ToLower()
if ($actual -ne $receipt.videoSha256) { throw 'Parent movie no longer matches its receipt' }
foreach ($dir in @('audio','capture','render')) { New-Item -ItemType Directory -Force ($env:TUTORIAL_OUTPUT+'/'+$dir) | Out-Null }
Copy-Item ($env:TUTORIAL_PARENT_OUTPUT+'/audio/*') ($env:TUTORIAL_OUTPUT+'/audio')
Copy-Item ($env:TUTORIAL_PARENT_OUTPUT+'/capture/*.json') ($env:TUTORIAL_OUTPUT+'/capture')
Copy-Item ($env:TUTORIAL_PARENT_OUTPUT+'/manifest.json') ($env:TUTORIAL_OUTPUT+'/parent-manifest.json')
Copy-Item ($env:TUTORIAL_PARENT_OUTPUT+'/edit-plan.json') ($env:TUTORIAL_OUTPUT+'/parent-edit-plan.json')
foreach ($id in @('install','run','results','beta','add','outro')) {
    foreach ($suffix in @('.video.mp4','.audio.wav','.mp4')) {
        Copy-Item ($env:TUTORIAL_PARENT_OUTPUT+'/render/'+$id+$suffix) ($env:TUTORIAL_OUTPUT+'/render/')
    }
}
Write-Output ('Revision initialized from parent SHA256 '+$actual)
