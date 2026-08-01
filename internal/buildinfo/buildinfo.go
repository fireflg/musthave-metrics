// Package buildinfo предоставляет единую точку для вывода информации о сборке.
package buildinfo

import "fmt"

// Переменные могут быть перезаписаны через -ldflags -X при сборке.
var (
	Version = "N/A"
	Date    = "N/A"
	Commit  = "N/A"
)

// Print выводит информацию о сборке в stdout.
func Print() {
	fmt.Printf("Build version: %s\n", Version)
	fmt.Printf("Build date: %s\n", Date)
	fmt.Printf("Build commit: %s\n", Commit)
}
