package utils

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func ColorizeLogs(logs []string) []string {

	for i, log := range logs {
		// Only style if not already styled (check for ANSI codes)
		if !strings.Contains(log, "\x1b[") {
			switch {
			case strings.Contains(log, "INFO"):
				logs[i] = strings.Replace(log, "INFO",
					lipgloss.NewStyle().
						Padding(0, 1, 0, 1).
						Bold(true).
						MaxWidth(80).
						Background(lipgloss.Color("87")).
						Foreground(lipgloss.Color("16")).
						Render("INFO"), 1)
			case strings.Contains(log, "ERRO"):
				logs[i] = strings.Replace(log, "ERRO",
					lipgloss.NewStyle().
						Padding(0, 1, 0, 1).
						Bold(true).
						MaxWidth(80).
						Background(lipgloss.Color("204")).
						Foreground(lipgloss.Color("0")).
						Render("ERRO"), 1)
			case strings.Contains(log, "DEBU"):
				logs[i] = strings.Replace(log, "DEBU",
					lipgloss.NewStyle().
						Padding(0, 1, 0, 1).
						Bold(true).
						MaxWidth(80).
						Background(lipgloss.Color("63")).
						Foreground(lipgloss.Color("0")).
						Render("DEBU"), 1)
			}
		}
	}
	return logs
}
