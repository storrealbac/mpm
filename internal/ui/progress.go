package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// WriteCounter counts the number of bytes written to it. It implements to the io.Writer interface
// and we can pass this into io.TeeReader() which will report progress on each write cycle.
type WriteCounter struct {
	Total uint64
	Read  uint64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Read += uint64(n)
	wc.PrintProgress()
	return n, nil
}

func (wc *WriteCounter) PrintProgress() {
	// Handle unknown content length
	if wc.Total == 0 || wc.Total > 1024*1024*1024*1024 { // >1TB likely invalid
		mbRead := float64(wc.Read) / 1024 / 1024
		status := InfoStyle.Render("↓")
		fmt.Printf("\r%s Downloading... %.1f MB", status, mbRead)
		return
	}

	// Calculate percentage
	percent := float64(wc.Read) / float64(wc.Total)
	if percent > 1.0 {
		percent = 1.0
	}

	// Create progress bar with gradient like multibar
	width := 20
	filled := int(percent * float64(width))

	// Build gradient progress bar
	var barStr string
	for i := 0; i < width; i++ {
		if i < filled {
			// Calculate position in the gradient (0.0 to 1.0)
			gradientPos := float64(i) / float64(width)

			// Smooth gradient
			color := interpolateColor(gradientPos)

			charStyle := lipgloss.NewStyle().Foreground(color)
			barStr += charStyle.Render("█")
		} else {
			emptyStyle := lipgloss.NewStyle().Foreground(grayColor)
			barStr += emptyStyle.Render("░")
		}
	}

	// Format file size
	mbRead := float64(wc.Read) / 1024 / 1024
	mbTotal := float64(wc.Total) / 1024 / 1024

	// Status indicator
	var status string
	if percent >= 1.0 {
		status = SuccessStyle.Render("✓")
	} else {
		status = InfoStyle.Render("↓")
	}

	percentText := fmt.Sprintf("%.1f%%", percent*100)
	sizeInfo := fmt.Sprintf("(%.1f/%.1f MB)", mbRead, mbTotal)

	// Style the text
	coloredPercent := lipgloss.NewStyle().Foreground(whiteColor).Bold(true).Render(percentText)
	coloredSize := lipgloss.NewStyle().Foreground(grayColor).Render(sizeInfo)

	// Add newline when complete
	if percent >= 1.0 {
		fmt.Printf("\r%s [%s] %s %s\n", status, barStr, coloredPercent, coloredSize)
	} else {
		fmt.Printf("\r%s [%s] %s %s", status, barStr, coloredPercent, coloredSize)
	}
}
