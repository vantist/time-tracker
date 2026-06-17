$ErrorActionPreference = 'Stop'

$repo = 'vantist/time-checker'
$artifact = 'tt-windows-amd64.exe'
$destDir = "$env:USERPROFILE\bin"
$dest = "$destDir\tt.exe"

$release = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest"
$tag = $release.tag_name
$url = "https://github.com/$repo/releases/download/$tag/$artifact"

Write-Host "Installing tt $tag..."
New-Item -ItemType Directory -Force -Path $destDir | Out-Null
Invoke-WebRequest -Uri $url -OutFile $dest

if ($env:PATH -notlike "*$destDir*") {
    Write-Host "Add $destDir to your PATH:"
    Write-Host '  [Environment]::SetEnvironmentVariable("PATH", $env:PATH + ";' + $destDir + '", "User")'
}

Write-Host "Installed: $( & $dest version )"
