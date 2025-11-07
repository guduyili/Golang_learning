# 在 PowerShell 中逐行执行以下命令：

go build -o server.exe

$p1 = Start-Process -FilePath ".\server.exe" -ArgumentList "-port=8001" -PassThru
$p2 = Start-Process -FilePath ".\server.exe" -ArgumentList "-port=8002" -PassThru
$p3 = Start-Process -FilePath ".\server.exe" -ArgumentList "-port=8003 -api=1" -PassThru

Start-Sleep 3

Write-Host "Testing..."

Invoke-RestMethod "http://localhost:4000/api?key=jw"
Invoke-RestMethod "http://localhost:4000/api?key=jw"
Invoke-RestMethod "http://localhost:4000/api?key=jw"

Write-Host "Press Enter to exit"
Read-Host

# Stop-Process -Id $p1.Id -Force
# Stop-Process -Id $p2.Id -Force
# Stop-Process -Id $p3.Id -Force

Remove-Item "server.exe"