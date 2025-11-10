# 在 PowerShell 中逐行执行以下命令：

go build -o server.exe

$p1 = Start-Process -FilePath ".\server.exe" -ArgumentList "-port=8001" -PassThru
$p2 = Start-Process -FilePath ".\server.exe" -ArgumentList "-port=8002" -PassThru
# 注意：将有空格的参数拆分为多个参数项，确保 -api 被正确解析
$p3 = Start-Process -FilePath ".\server.exe" -ArgumentList "-port=8003", "-api=true" -PassThru

Start-Sleep 3

Write-Host "Testing..."


# $jobs = @(
#   Start-Job { Invoke-RestMethod "http://localhost:4000/api?key=jw" }
#   Start-Job { Invoke-RestMethod "http://localhost:4000/api?key=jw" }
#   Start-Job { Invoke-RestMethod "http://localhost:4000/api?key=jw" }
# )

# # 等待全部完成并输出结果
# Receive-Job -Job $jobs -Wait -AutoRemoveJob | ForEach-Object { $_ }

$ps = @(
  Start-Process curl.exe -ArgumentList "http://localhost:4000/api?key=jw" -NoNewWindow -PassThru
  Start-Process curl.exe -ArgumentList "http://localhost:4000/api?key=jw" -NoNewWindow -PassThru
  Start-Process curl.exe -ArgumentList "http://localhost:4000/api?key=jw" -NoNewWindow -PassThru
)

Wait-Process -Id ($ps.Id)

Write-Host "Press Enter to exit"
Read-Host

# 先停止进程，再删除可执行文件；否则文件被占用会导致删除失败
Stop-Process -Id $p1.Id -Force -ErrorAction SilentlyContinue
Stop-Process -Id $p2.Id -Force -ErrorAction SilentlyContinue
Stop-Process -Id $p3.Id -Force -ErrorAction SilentlyContinue

# 等待进程完全退出，避免文件仍被占用
if ($p1 -and (Get-Process -Id $p1.Id -ErrorAction SilentlyContinue)) { Wait-Process -Id $p1.Id -Timeout 5 -ErrorAction SilentlyContinue }
if ($p2 -and (Get-Process -Id $p2.Id -ErrorAction SilentlyContinue)) { Wait-Process -Id $p2.Id -Timeout 5 -ErrorAction SilentlyContinue }
if ($p3 -and (Get-Process -Id $p3.Id -ErrorAction SilentlyContinue)) { Wait-Process -Id $p3.Id -Timeout 5 -ErrorAction SilentlyContinue }

Remove-Item "server.exe" -Force -ErrorAction SilentlyContinue