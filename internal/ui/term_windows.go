//go:build windows

package ui

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/AliHamza-Coder/crush/internal/fileutil"
)

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procGetStdHandle     = kernel32.NewProc("GetStdHandle")
	procSetConsoleMode   = kernel32.NewProc("SetConsoleMode")
	procGetConsoleMode   = kernel32.NewProc("GetConsoleMode")
	procReadConsoleInput = kernel32.NewProc("ReadConsoleInputW")
)

// enableLineInput disables line input, enableEchoInput disables echo, enableVTProcessing enables ANSI escape sequences.


const (
	stdInputHandle  = uintptr(0xfffffff5)
	keyEvent        = 0x0001
	vkUp            = 0x26
	vkDown          = 0x28
	vkReturn        = 0x0D
	vkEscape        = 0x1B
	enableLineInput = 0x0002
	enableEchoInput = 0x0004
	enableVTProcessing = 0x0004 // ENABLE_VIRTUAL_TERMINAL_PROCESSING
)

type inputRecord struct {
	eventType uint16
	_         [2]byte
	event     [16]byte
}

type keyEventRecord struct {
	keyDown         int32
	repeatCount     uint16
	virtualKeyCode  uint16
	virtualScanCode uint16
	unicodeChar     uint16
	controlKeyState uint32
}

func readOneKey() (rune, uint16, bool) {
	handle, _, _ := procGetStdHandle.Call(stdInputHandle)
	var oldMode uint32
	procGetConsoleMode.Call(handle, uintptr(unsafe.Pointer(&oldMode)))
	// Disable line input and echo, then enable virtual terminal processing for ANSI escape support
	newMode := oldMode & ^uint32(enableLineInput|enableEchoInput)
	newMode |= uint32(enableVTProcessing)
	procSetConsoleMode.Call(handle, uintptr(newMode))
	defer procSetConsoleMode.Call(handle, uintptr(oldMode))

	var rec inputRecord
	var read uint32
	for {
		ret, _, _ := procReadConsoleInput.Call(handle, uintptr(unsafe.Pointer(&rec)), 1, uintptr(unsafe.Pointer(&read)))
		if ret == 0 || read == 0 {
			return 0, 0, false
		}
		if rec.eventType != keyEvent {
			continue
		}
		ke := (*keyEventRecord)(unsafe.Pointer(&rec.event))
		if ke.keyDown == 0 {
			continue
		}
		return rune(ke.unicodeChar), ke.virtualKeyCode, true
	}
}


func countLines(items []string) int {
	return len(items)
}

func SelectFromList(items []string, prompt string) string {
	selected := 0
	firstRender := true

	for {
		if firstRender {
			// Initial render prints prompt and items
			fmt.Printf("\n")
			fmt.Printf("  %s\n", prompt)
			firstRender = false
		} else {
			// Clear previously printed block (prompt + items)
			clearLines(countLines(items) + 2)
		}

		for i, item := range items {
			if i == selected {
				prefix := fmt.Sprintf("  %s>%s ", fileutil.Green, fileutil.Reset)
				mark := ""
				if i == 0 {
					mark = fmt.Sprintf("  %s(default)%s", fileutil.Dim, fileutil.Reset)
				}
				fmt.Printf("%s%s%s%s\n", prefix, fileutil.Bold, item, mark)
			} else {
				mark := ""
				if i == 0 {
					mark = fmt.Sprintf("  %s(default)%s", fileutil.Dim, fileutil.Reset)
				}
				fmt.Printf("   %s%s\n", item, mark)
			}
		}

		char, vk, ok := readOneKey()
		if !ok {
			return items[0]
		}

		switch {
		case vk == vkUp:
			if selected > 0 {
				selected--
			}
		case vk == vkDown:
			if selected < len(items)-1 {
				selected++
			}
		case vk == vkReturn || char == '\r':
			fmt.Printf("\n")
			return items[selected]
		case vk == vkEscape || char == 'q' || char == 'Q':
			fmt.Printf("\n")
			return ""
		case char >= '1' && char <= '9':
			 n := int(char - '1')
			 if n < len(items) {
				 fmt.Printf("\n")
				 return items[n]
			 }
		}
	}
}

func moveUp(lines int) {
	if lines > 0 {
		fmt.Printf("\033[%dA", lines)
	}
}

// clearLines erases the given number of lines above the cursor.
func clearLines(lines int) {
	for i := 0; i < lines; i++ {
		fmt.Print("\r\033[K")
		if i < lines-1 {
			fmt.Print("\033[1A")
		}
	}
}
