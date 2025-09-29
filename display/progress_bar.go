package display

import (
	"fmt"
	"time"
)

var spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
var spinnerIndex = 0

func getSpinner() string {
	char := spinnerChars[spinnerIndex]
	spinnerIndex = (spinnerIndex + 1) % len(spinnerChars)
	return char
}

// IndexingProgress represents indexing progress data
type IndexingProgress struct {
	ProcessedFiles int
	TotalFiles     int
	FunctionsFound int
	TypesFound     int
	ElapsedTime    time.Duration
	FilesPerSecond float64
}

// ShowIndexingProgress displays dynamic indexing progress
func ShowIndexingProgress(progress IndexingProgress) {
	percentage := float64(progress.ProcessedFiles) / float64(progress.TotalFiles) * 100
	filesPerSec := float64(progress.ProcessedFiles) / progress.ElapsedTime.Seconds()

	fmt.Printf("\r%s Indexing: %.1f%% (%d/%d files, %.1f files/sec, %d functions, %d types)",
		getSpinner(), percentage, progress.ProcessedFiles, progress.TotalFiles, filesPerSec,
		progress.FunctionsFound, progress.TypesFound)
}

// ShowIndexingStart displays the start message
func ShowIndexingStart() {
	fmt.Println("🔄 Starting code indexing...")
}

// ShowIndexingComplete displays completion message
func ShowIndexingComplete() {
	fmt.Println() // New line after progress
	fmt.Println("✅ Indexing completed!")
	fmt.Println()
}
