package ui

import (
	"os"

	"github.com/manifoldco/promptui"
)

type selectOption struct {
	Name      string
	IsDefault bool
}

func SelectFromList(items []string, prompt string) string {
	if len(items) == 0 {
		return ""
	}

	options := make([]selectOption, len(items))
	for i, item := range items {
		options[i] = selectOption{
			Name:      item,
			IsDefault: i == 0,
		}
	}

	templates := &promptui.SelectTemplates{
		Label:    "  {{ . | bold }}",
		Active:   `  {{ "> " | green }}{{ .Name | bold }}{{ if .IsDefault }}  {{ "(default)" | dim }}{{ end }}`,
		Inactive: `    {{ .Name }}{{ if .IsDefault }}  {{ "(default)" | dim }}{{ end }}`,
		Selected: `  {{ "✔" | green }} {{ .Name | bold }}`,
	}

	promptSelect := promptui.Select{
		Label:        prompt,
		Items:        options,
		Templates:    templates,
		Size:         len(items),
		HideHelp:     true,
		HideSelected: false,
	}

	index, _, err := promptSelect.Run()
	if err != nil {
		if err == promptui.ErrInterrupt || err == promptui.ErrEOF {
			os.Exit(0)
		}
		return ""
	}

	return items[index]
}
