package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"corexs-designer/internal/bridge"

	webview "github.com/jchv/go-webview2"
)

//go:embed assets/editor.html
var editorHTML string

//go:embed assets/corexs.ico
var iconData []byte

func main() {
	runtime.LockOSThread()

	w := webview.NewWithOptions(webview.WebViewOptions{
		Debug:     false,
		AutoFocus: true,
	})
	if w == nil {
		panic("WebView2 could not be created.\nInstall: https://developer.microsoft.com/en-us/microsoft-edge/webview2/")
	}
	defer w.Destroy()

	w.SetTitle("CorexS Designer v2026.1")
	w.SetSize(1440, 900, webview.HintNone)

	// Set window icon (taskbar + title bar)
	setWindowIcon(w)

	var b *bridge.Bridge
	b = bridge.New(
		func(js string) {
			w.Dispatch(func() { w.Eval(js) })
		},
		func(filter string) string { return openDialog(filter) },
		func(name, filter string) string { return saveDialog(name, filter) },
	)

	w.Bind("__goBridge", func(raw string) string {
		b.Handle(raw)
		return ""
	})

	w.Init(`
		window.__corexsPostMessage = function(msg) {
			__goBridge(typeof msg === 'string' ? msg : JSON.stringify(msg));
		};
	`)

	w.SetHtml(editorHTML)
	w.Run()
}

// ── Window Icon ───────────────────────────────────────────────────────────────

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procSendMessage              = user32.NewProc("SendMessageW")
	procCreateIconFromResourceEx = user32.NewProc("CreateIconFromResourceEx")
)

const (
	wmSetIcon       = 0x0080
	iconSmall       = 0
	iconBig         = 1
	LR_DEFAULTCOLOR = 0x0000
)

func setWindowIcon(_ interface{}) {
	// Write ico to temp file, load as HICON via LR_LOADFROMFILE
	tmp := os.TempDir() + "\\corexs_icon.ico"
	if err := os.WriteFile(tmp, iconData, 0644); err != nil {
		return
	}

	loadImage := user32.NewProc("LoadImageW")
	p, _ := syscall.UTF16PtrFromString(tmp)

	// Load large icon (32x32)
	hIconBig, _, _ := loadImage.Call(
		0,
		uintptr(unsafe.Pointer(p)),
		1,   // IMAGE_ICON
		256, // width — Windows will pick best match
		256,
		0x00000010, // LR_LOADFROMFILE
	)

	// Load small icon (16x16)
	hIconSmall, _, _ := loadImage.Call(
		0,
		uintptr(unsafe.Pointer(p)),
		1,
		16,
		16,
		0x00000010,
	)

	// Give Windows a moment to create the window, then set icon
	// We'll do it via a goroutine with a small delay approach,
	// but since LockOSThread is set we use a timer approach via SetTimer
	// Actually: call after SetHtml by hooking into run loop isn't easy.
	// Simpler: find the hwnd via GetForegroundWindow after webview starts.
	// We'll use a background goroutine via channels.

	if hIconBig != 0 {
		hwnd, _, _ := procGetForegroundWindow.Call()
		if hwnd != 0 {
			procSendMessage.Call(hwnd, wmSetIcon, iconBig, hIconBig)
			procSendMessage.Call(hwnd, wmSetIcon, iconSmall, hIconSmall)
		}
	}
}

// ── Windows File Dialogs ──────────────────────────────────────────────────────

var (
	comdlg32            = syscall.NewLazyDLL("comdlg32.dll")
	procGetOpenFileName = comdlg32.NewProc("GetOpenFileNameW")
	procGetSaveFileName = comdlg32.NewProc("GetSaveFileNameW")
)

type ofnW struct {
	lStructSize       uint32
	hwndOwner         uintptr
	hInstance         uintptr
	lpstrFilter       *uint16
	lpstrCustomFilter *uint16
	nMaxCustFilter    uint32
	nFilterIndex      uint32
	lpstrFile         *uint16
	nMaxFile          uint32
	lpstrFileTitle    *uint16
	nMaxFileTitle     uint32
	lpstrInitialDir   *uint16
	lpstrTitle        *uint16
	flags             uint32
	nFileOffset       uint16
	nFileExtension    uint16
	lpstrDefExt       *uint16
	lCustData         uintptr
	lpfnHook          uintptr
	lpTemplateName    *uint16
	pvReserved        unsafe.Pointer
	dwReserved        uint32
	flagsEx           uint32
}

func u16p(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }

func openDialog(filter string) string {
	buf := make([]uint16, 2048)
	o := ofnW{
		lStructSize: uint32(unsafe.Sizeof(ofnW{})),
		lpstrFilter: u16p(filter),
		lpstrTitle:  u16p("Open File — CorexS Designer"),
		lpstrFile:   &buf[0],
		nMaxFile:    uint32(len(buf)),
		flags:       0x00001000 | 0x00000800 | 0x00000008 | 0x00080000,
	}
	r, _, _ := procGetOpenFileName.Call(uintptr(unsafe.Pointer(&o)))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

func saveDialog(defaultName, filter string) string {
	buf := make([]uint16, 2048)
	n, _ := syscall.UTF16FromString(defaultName)
	copy(buf, n)
	// Detect default extension from filename
	defExt := "css"
	if strings.HasSuffix(defaultName, ".html") {
		defExt = "html"
	} else if strings.HasSuffix(defaultName, ".corexsd") {
		defExt = "corexsd"
	}
	o := ofnW{
		lStructSize: uint32(unsafe.Sizeof(ofnW{})),
		lpstrFilter: u16p(filter),
		lpstrTitle:  u16p("Save File — CorexS Designer"),
		lpstrDefExt: u16p(defExt),
		lpstrFile:   &buf[0],
		nMaxFile:    uint32(len(buf)),
		flags:       0x00000002 | 0x00000800 | 0x00000008 | 0x00080000,
	}
	r, _, _ := procGetSaveFileName.Call(uintptr(unsafe.Pointer(&o)))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func jsonStr(v interface{}) string { b, _ := json.Marshal(v); return string(b) }

func absPath(rel string) string {
	abs, err := filepath.Abs(rel)
	if err != nil {
		return rel
	}
	return abs
}

func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("cannot read: %w", err)
	}
	return string(data), nil
}
