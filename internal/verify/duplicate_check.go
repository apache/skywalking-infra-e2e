package verify

import (
    "fmt"
)

// CheckForDuplicates verifies if there are duplicate entries in a list.
// It returns an error if any duplicates are found.
func CheckForDuplicates(list []string) error {
    seen := make(map[string]bool)
    for _, item := range list {
        if seen[item] {
            return fmt.Errorf("duplicate data found: %s", item)
        }
        seen[item] = true
    }
    return nil
}

