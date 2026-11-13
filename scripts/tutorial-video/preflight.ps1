Write-Output 'TOOLS'
Get-Command ffmpeg,ffprobe,scoop,kspeeder,ks-cli,python,node,npm,ssh -ErrorAction SilentlyContinue | Select-Object Name,Source | ConvertTo-Json -Compress
foreach($name in @('TUTORIAL_SERVER_ROOT','TUTORIAL_FFMPEG','TUTORIAL_FFPROBE','TUTORIAL_VOICE_REFERENCE')) {
  $value=[Environment]::GetEnvironmentVariable($name)
  if(-not $value -or -not(Test-Path $value)) { throw "Missing configured resource: $name" }
  Write-Output "$name OK"
}
Write-Output 'SPEECH'
$models=$env:TUTORIAL_TTS_ENDPOINT.Replace('/audio/speech','/models')
(Invoke-WebRequest -UseBasicParsing $models -TimeoutSec 5).StatusCode
Set-Location ($env:TUTORIAL_SERVER_ROOT+'/tool')
node --version
node health.mjs
if($LASTEXITCODE -ne 0) { throw 'Browser health check failed' }
