//go:build !windows

package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/AliHamza-Coder/crush/internal/fileutil"
)

func SelectFromList(items []string, prompt string) string {
	for {
		fmt.Printf("\n")
		fmt.Printf("  %s\n", prompt)
		for i, item := range items {
			mark := ""
			if i == 0 {
				mark = fmt.Sprintf("  %s★ recommended%s", fileutil.Dim, fileutil.Reset)
			}
			fmt.Printf("  %s[%d]%s %s%s\n", fileutil.Bold, i+1, fileutil.Reset, item, mark)
		}
		fmt.Printf("\n")
		input := ReadInput("  Number (or type name): ")
		if input == "" {
			return ""
		}
		if n, err := strconv.Atoi(input); err == nil && n >= 1 && n <= len(items) {
			return items[n-1]
		}
		input = strings.ToLower(strings.TrimSpace(input))
		for _, item := range items {
			if item == input {
				return input
			}
		}
		Beep()
		fmt.Printf("  %sInvalid. Enter 1-%d or format name.%s\n", fileutil.Red, len(items), fileutil.Reset)
	}
}
