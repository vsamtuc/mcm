package greet

import "fmt"

// Hello returns a friendly greeting.
// This lives in pkg/ because it's OK for other modules to import.
func Hello(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}
