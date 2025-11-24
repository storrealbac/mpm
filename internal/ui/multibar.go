package ui

import (
	"fmt"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

// MultiProgressBar manages multiple progress bars displayed on separate lines
type MultiProgressBar struct {
	bars   map[int]*ProgressBar
	mutex  sync.Mutex
	maxID  int
	active bool
}

type ProgressBar struct {
	ID       int
	Name     string
	Total    uint64
	Current  uint64
	Finished bool
}

var globalMultiBar *MultiProgressBar

// InitMultiBar initializes the global multi-progress bar
func InitMultiBar() {
	globalMultiBar = &MultiProgressBar{
		bars:   make(map[int]*ProgressBar),
		active: true,
	}
}

// AddBar creates a new progress bar and returns its ID
func AddBar(name string, total uint64) int {
	if globalMultiBar == nil {
		return -1
	}

	globalMultiBar.mutex.Lock()
	defer globalMultiBar.mutex.Unlock()

	globalMultiBar.maxID++
	id := globalMultiBar.maxID

	globalMultiBar.bars[id] = &ProgressBar{
		ID:       id,
		Name:     name,
		Total:    total,
		Current:  0,
		Finished: false,
	}

	// Print initial line for this bar
	fmt.Println()

	return id
}

// UpdateBar updates the progress of a specific bar
func UpdateBar(id int, current uint64) {
	if globalMultiBar == nil {
		return
	}

	globalMultiBar.mutex.Lock()
	defer globalMultiBar.mutex.Unlock()

	bar, exists := globalMultiBar.bars[id]
	if !exists {
		return
	}

	bar.Current = current
	globalMultiBar.render()
}

// SetBarTotal updates the total size of a bar
func SetBarTotal(id int, total uint64) {
	if globalMultiBar == nil {
		return
	}

	globalMultiBar.mutex.Lock()
	defer globalMultiBar.mutex.Unlock()

	bar, exists := globalMultiBar.bars[id]
	if !exists {
		return
	}

	bar.Total = total
	globalMultiBar.render()
}

// FinishBar marks a bar as finished
func FinishBar(id int) {
	if globalMultiBar == nil {
		return
	}

	globalMultiBar.mutex.Lock()
	defer globalMultiBar.mutex.Unlock()

	bar, exists := globalMultiBar.bars[id]
	if !exists {
		return
	}

	bar.Finished = true
	bar.Current = bar.Total
	globalMultiBar.render()
}

// CloseMultiBar closes the multi-progress display
func CloseMultiBar() {
	if globalMultiBar == nil {
		return
	}

	globalMultiBar.mutex.Lock()
	defer globalMultiBar.mutex.Unlock()

	globalMultiBar.active = false
	// Move cursor to bottom
	fmt.Print("\n")
}

// render displays all progress bars (caller must hold mutex)
func (m *MultiProgressBar) render() {
	if !m.active {
		return
	}

	// Count active bars to know how many lines to update
	numBars := len(m.bars)
	if numBars == 0 {
		return
	}

	// Move cursor up by the number of bars
	for i := 0; i < numBars; i++ {
		fmt.Print("\033[A") // Move up one line
	}

	// Render each bar
	for i := 1; i <= m.maxID; i++ {
		bar, exists := m.bars[i]
		if !exists {
			continue
		}

		// Clear line
		fmt.Print("\033[2K\r")

		// Calculate percentage
		percent := float64(0)
		if bar.Total > 0 {
			percent = float64(bar.Current) / float64(bar.Total)
			if percent > 1.0 {
				percent = 1.0
			}
		}

		// Create progress bar with gradient
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

		// Format size
		mbCurrent := float64(bar.Current) / 1024 / 1024
		mbTotal := float64(bar.Total) / 1024 / 1024

		status := ""
		if bar.Finished {
			status = SuccessStyle.Render("✓")
		} else {
			status = InfoStyle.Render("↓")
		}

		// Truncate name if too long
		displayName := bar.Name
		if len(displayName) > 30 {
			displayName = displayName[:27] + "..."
		}

		if bar.Total > 0 {
			fmt.Printf("%s %-30s [%s] %5.1f%% (%.1f/%.1f MB)\n",
				status,
				displayName,
				barStr,
				percent*100,
				mbCurrent,
				mbTotal,
			)
		} else {
			fmt.Printf("%s %-30s [%s] %.1f MB\n",
				status,
				displayName,
				barStr,
				mbCurrent,
			)
		}
	}
}

// interpolateColor creates a smooth gradient from blue to cyan
func interpolateColor(t float64) lipgloss.Color {
	// Smooth gradient: Blue -> Cyan
	// Blue: #3B82F6 (59, 130, 246)
	// Cyan: #06B6D4 (6, 182, 212)

	r := lerp(59, 6, t)
	g := lerp(130, 182, t)
	b := lerp(246, 212, t)

	// Convert to hex
	return lipgloss.Color(rgbToHex(int(r), int(g), int(b)))
}

// lerp performs linear interpolation between two values
func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

// rgbToHex converts RGB to hex color string
func rgbToHex(r, g, b int) string {
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}
