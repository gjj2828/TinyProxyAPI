package tray

import (
	"syscall"
	"testing"
	"unsafe"

	"github.com/lxn/win"
)

// TestIconicStateSurvivesHideShow 锁定两个 bug 根因依赖的 Win32 语义。
//
// 走纯 Win32 而不经过 walk：walk 的窗口只能在消息循环（FormBase.Run）启动后
// 才能构造，单测进程里拿不到可用的日志窗口。但下面这三条都是系统行为，
// 与 walk 无关，直接断言比在注释里引用文档更可靠——修复完全建立在这三条之上。
//
//  1. SW_HIDE 不清除 iconic 标志 → 最小化后再隐藏，窗口仍留在 iconic 状态
//  2. SW_SHOWNA 不还原最小化窗口 → walk 的 Show 内部正是 SW_SHOWNA
//     （见 walk/window.go 的 setWindowVisible），所以唤不回来
//  3. SW_RESTORE 同时显示并还原 → LogWindow.Show 修复所依赖的调用
//
// 窗口放在屏幕外，测试期间不会在桌面上闪现。
func TestIconicStateSurvivesHideShow(t *testing.T) {
	className := utf16Ptr(t, "TinyProxyAPIIconicProbe")
	title := utf16Ptr(t, "probe")

	var wc win.WNDCLASSEX
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	wc.LpfnWndProc = syscall.NewCallback(probeWndProc)
	wc.HInstance = win.GetModuleHandle(nil)
	wc.LpszClassName = className
	if win.RegisterClassEx(&wc) == 0 {
		t.Fatal("RegisterClassEx failed")
	}

	hwnd := win.CreateWindowEx(
		0, className, title, win.WS_OVERLAPPEDWINDOW,
		-3000, -3000, 100, 100, 0, 0, wc.HInstance, nil)
	if hwnd == 0 {
		t.Fatal("CreateWindowEx failed")
	}
	defer win.DestroyWindow(hwnd)

	win.ShowWindow(hwnd, win.SW_SHOW)
	if win.IsIconic(hwnd) {
		t.Fatal("freshly shown window is iconic")
	}

	win.ShowWindow(hwnd, win.SW_MINIMIZE)
	if !win.IsIconic(hwnd) {
		t.Fatal("SW_MINIMIZE did not set the iconic state")
	}

	// 1. SW_HIDE 不清除 iconic——根因所在
	win.ShowWindow(hwnd, win.SW_HIDE)
	if !win.IsIconic(hwnd) {
		t.Fatal("SW_HIDE cleared the iconic state")
	}

	// 2. SW_SHOWNA 不还原——walk.Show 走这条路径，故"点显示日志没反应"
	win.ShowWindow(hwnd, win.SW_SHOWNA)
	if !win.IsIconic(hwnd) {
		t.Fatal("SW_SHOWNA restored the iconic window")
	}

	// 3. SW_RESTORE 才能同时显示并还原
	win.ShowWindow(hwnd, win.SW_HIDE)
	win.ShowWindow(hwnd, win.SW_RESTORE)
	if win.IsIconic(hwnd) {
		t.Fatal("SW_RESTORE did not clear the iconic state")
	}
	if !win.IsWindowVisible(hwnd) {
		t.Fatal("SW_RESTORE did not show the window")
	}
}

// probeWndProc 把消息原样交还缺省窗口过程；测试只关心 ShowWindow 的状态位。
func probeWndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func utf16Ptr(t *testing.T, s string) *uint16 {
	t.Helper()
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		t.Fatalf("UTF16PtrFromString(%q): %v", s, err)
	}
	return p
}
