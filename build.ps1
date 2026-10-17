# Build script for TinyProxyAPI
# Uses -ldflags="-H windowsgui" to hide console window on Windows

$ErrorActionPreference = "Stop"

Write-Host "Building TinyProxyAPI..." -ForegroundColor Cyan

go build -ldflags="-H windowsgui" -o TinyProxyAPI.exe

if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed with exit code: $LASTEXITCODE" -ForegroundColor Red
    exit $LASTEXITCODE
}

# 剥掉继承自工作区目录的 Low 完整性标签。
#
# 为什么必须做：DSH 沙箱会给工作区目录打 `Mandatory Label\Low Mandatory Level:(OI)(CI)(NW)`，
# 且该标签是常驻的（DSH 文档：a standing Low label outlives DSH ... survives the session）。
# 新建的 exe 会继承这个标签，而 Windows 规定进程实际完整性级别 = min(用户 IL, 文件 IL)，
# 于是 exe 无论从哪启动都会被压到 Low IL。Low IL 进程的窗口消息受 UIPI 限制，
# 不能发给 explorer 的通知区域，Shell_NotifyIcon(NIM_ADD) 必然返回 FALSE 且
# GetLastError() == 5 (ERROR_ACCESS_DENIED) —— 表现就是启动即报「托盘创建失败」。
#
# 只改这一个文件的 SACL，不动目录、不动 DACL，因此不影响 DSH 沙箱的写入边界。
# icacls 官方语义：/setintegritylevel 只添加/替换完整性 ACE，不触碰 DACL
# （https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/icacls）。
#
# 必须用 Medium 及以上 IL 的进程执行。DSH 的 workspace-write 沙箱会把工具进程降到
# Low IL，此时 icacls 会报 Access is denied —— 所以本脚本要在普通 PowerShell 里跑，
# 不能经由 DSH 工具通道调用。
icacls TinyProxyAPI.exe /setintegritylevel M | Out-Null
if ($LASTEXITCODE -ne 0) {
    Write-Host "WARNING: failed to reset integrity level; the tray icon will fail to create." -ForegroundColor Yellow
    Write-Host "         Run this script from a normal PowerShell window (not the DSH sandbox)." -ForegroundColor Yellow
} else {
    Write-Host "Integrity level reset to Medium (tray icon will work)." -ForegroundColor DarkGray
}

Write-Host "Build successful: TinyProxyAPI.exe" -ForegroundColor Green
