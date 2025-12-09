//go:build windows

// Package dialog provides simple input dialogs for Windows using native Win32 API
package dialog

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procCreateWindowExW      = user32.NewProc("CreateWindowExW")
	procDefWindowProcW       = user32.NewProc("DefWindowProcW")
	procDestroyWindow        = user32.NewProc("DestroyWindow")
	procDispatchMessageW     = user32.NewProc("DispatchMessageW")
	procGetMessageW          = user32.NewProc("GetMessageW")
	procGetWindowTextW       = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW = user32.NewProc("GetWindowTextLengthW")
	procPostQuitMessage      = user32.NewProc("PostQuitMessage")
	procRegisterClassExW     = user32.NewProc("RegisterClassExW")

	procSetFocus = user32.NewProc("SetFocus")

	procShowWindow       = user32.NewProc("ShowWindow")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procUpdateWindow     = user32.NewProc("UpdateWindow")
	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")

	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
)

//goland:noinspection GoUnusedConst,GoSnakeCaseUsage
const (
	WS_OVERLAPPED       = 0x00000000
	WS_CAPTION          = 0x00C00000
	WS_SYSMENU          = 0x00080000
	WS_VISIBLE          = 0x10000000
	WS_CHILD            = 0x40000000
	WS_TABSTOP          = 0x00010000
	WS_BORDER           = 0x00800000
	WS_EX_DLGMODALFRAME = 0x00000001
	WS_EX_TOPMOST       = 0x00000008

	ES_PASSWORD    = 0x0020
	ES_AUTOHSCROLL = 0x0080

	BS_DEFPUSHBUTTON = 0x0001
	BS_PUSHBUTTON    = 0x0000

	SS_LEFT = 0x0000

	WM_CREATE   = 0x0001
	WM_DESTROY  = 0x0002
	WM_CLOSE    = 0x0010
	WM_COMMAND  = 0x0111
	WM_SETFONT  = 0x0030
	WM_SETFOCUS = 0x0007

	SW_SHOW = 5

	SM_CXSCREEN = 0
	SM_CYSCREEN = 1

	SWP_NOSIZE   = 0x0001
	SWP_NOZORDER = 0x0004

	MB_OK        = 0x00000000
	MB_ICONINFO  = 0x00000040
	MB_ICONERROR = 0x00000010
	MB_TOPMOST   = 0x00040000

	ID_OK     = 1
	ID_CANCEL = 2
	ID_EDIT   = 100
)

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     syscall.Handle
	HIcon         syscall.Handle
	HCursor       syscall.Handle
	HbrBackground syscall.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       syscall.Handle
}

type MSG struct {
	Hwnd    syscall.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

// Dialog state for the callback
type dialogState struct {
	hwndEdit syscall.Handle
	result   string
	ok       bool
	masked   bool
	prompt   string
}

var (
	currentDialog *dialogState
	dialogMu      sync.Mutex
	classCounter  int
)

// InputBox shows an input dialog and returns the entered text.
// This function is thread-safe and can be called from any goroutine.
func InputBox(title, prompt string, masked bool) (string, bool) {
	resultChan := make(chan struct {
		text string
		ok   bool
	}, 1)

	// Run dialog on a dedicated OS thread
	go func() {
		// Lock this goroutine to its current OS thread
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		text, ok := showInputBoxOnThread(title, prompt, masked)
		resultChan <- struct {
			text string
			ok   bool
		}{text, ok}
	}()

	result := <-resultChan
	return result.text, result.ok
}

// showInputBoxOnThread must be called from a thread locked with LockOSThread
func showInputBoxOnThread(title, prompt string, masked bool) (string, bool) {
	dialogMu.Lock()
	classCounter++
	classNum := classCounter
	dialogMu.Unlock()

	state := &dialogState{
		masked: masked,
		prompt: prompt,
	}

	dialogMu.Lock()
	currentDialog = state
	dialogMu.Unlock()

	defer func() {
		dialogMu.Lock()
		currentDialog = nil
		dialogMu.Unlock()
	}()

	// Use unique class name to avoid conflicts
	classNameStr := fmt.Sprintf("SteelClockInputBox%d", classNum)
	className := utf16Ptr(classNameStr)
	hInstance, _, _ := procGetModuleHandleW.Call(0)

	// Register window class
	wc := WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		LpfnWndProc:   syscall.NewCallback(inputBoxWndProc),
		HInstance:     syscall.Handle(hInstance),
		HbrBackground: 16, // COLOR_BTNFACE + 1
		LpszClassName: className,
	}

	_, _, _ = procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	// Window dimensions
	width := int32(350)
	height := int32(140)

	// Center on screen
	screenW, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	screenH, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
	x := (int32(screenW) - width) / 2
	y := (int32(screenH) - height) / 2

	// Create window
	hwnd, _, _ := procCreateWindowExW.Call(
		WS_EX_DLGMODALFRAME|WS_EX_TOPMOST,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16Ptr(title))),
		WS_OVERLAPPED|WS_CAPTION|WS_SYSMENU|WS_VISIBLE,
		uintptr(x), uintptr(y), uintptr(width), uintptr(height),
		0, 0, hInstance, 0,
	)

	if hwnd == 0 {
		return "", false
	}

	_, _, _ = procShowWindow.Call(hwnd, SW_SHOW)
	_, _, _ = procUpdateWindow.Call(hwnd)

	// Bring to foreground
	_, _, _ = procSetForegroundWindow.Call(hwnd)

	// Focus on edit control
	dialogMu.Lock()
	hwndEdit := state.hwndEdit
	dialogMu.Unlock()
	if hwndEdit != 0 {
		_, _, _ = procSetFocus.Call(uintptr(hwndEdit))
	}

	// Message loop
	var msg MSG
	for {
		ret, _, _ := procGetMessageW.Call(
			uintptr(unsafe.Pointer(&msg)),
			0, 0, 0,
		)
		if ret == 0 || int32(ret) == -1 {
			break
		}
		_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		_, _, _ = procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}

	dialogMu.Lock()
	result := state.result
	ok := state.ok
	dialogMu.Unlock()

	return result, ok
}

func inputBoxWndProc(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		hInstance, _, _ := procGetModuleHandleW.Call(0)

		// Create label
		_, _, _ = procCreateWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(utf16Ptr("STATIC"))),
			uintptr(unsafe.Pointer(utf16Ptr(currentDialog.prompt))),
			WS_CHILD|WS_VISIBLE|SS_LEFT,
			10, 10, 320, 20,
			uintptr(hwnd), 0, hInstance, 0,
		)

		// Create edit box
		editStyle := uintptr(WS_CHILD | WS_VISIBLE | WS_BORDER | WS_TABSTOP | ES_AUTOHSCROLL)
		if currentDialog.masked {
			editStyle |= ES_PASSWORD
		}

		hwndEdit, _, _ := procCreateWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(utf16Ptr("EDIT"))),
			0,
			editStyle,
			10, 35, 315, 22,
			uintptr(hwnd), ID_EDIT, hInstance, 0,
		)
		currentDialog.hwndEdit = syscall.Handle(hwndEdit)

		// Create OK button
		_, _, _ = procCreateWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(utf16Ptr("BUTTON"))),
			uintptr(unsafe.Pointer(utf16Ptr("OK"))),
			WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_DEFPUSHBUTTON,
			160, 70, 75, 25,
			uintptr(hwnd), ID_OK, hInstance, 0,
		)

		// Create Cancel button
		_, _, _ = procCreateWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(utf16Ptr("BUTTON"))),
			uintptr(unsafe.Pointer(utf16Ptr("Cancel"))),
			WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,
			250, 70, 75, 25,
			uintptr(hwnd), ID_CANCEL, hInstance, 0,
		)

		return 0

	case WM_COMMAND:
		id := int(wParam & 0xFFFF)
		switch id {
		case ID_OK:
			// Get text from edit control
			currentDialog.result = getWindowText(currentDialog.hwndEdit)
			currentDialog.ok = true
			_, _, _ = procDestroyWindow.Call(uintptr(hwnd))
		case ID_CANCEL:
			currentDialog.ok = false
			_, _, _ = procDestroyWindow.Call(uintptr(hwnd))
		}
		return 0

	case WM_CLOSE:
		currentDialog.ok = false
		_, _, _ = procDestroyWindow.Call(uintptr(hwnd))
		return 0

	case WM_DESTROY:
		_, _, _ = procPostQuitMessage.Call(0)
		return 0
	}

	ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

func getWindowText(hwnd syscall.Handle) string {
	length, _, _ := procGetWindowTextLengthW.Call(uintptr(hwnd))
	if length == 0 {
		return ""
	}

	buf := make([]uint16, length+1)
	_, _, _ = procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), length+1)

	return strings.TrimRight(string(utf16.Decode(buf)), "\x00")
}

func utf16Ptr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}
