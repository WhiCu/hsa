package errkit

import (
	"fmt"
	"strings"
)

type ErrorFormatFunc func([]error) string

// ⚡ Bolt: Using strings.Builder instead of slice allocation + strings.Join to reduce memory allocations and improve performance
func ListFormatFunc(es []error) string {
	if len(es) == 1 {
		return fmt.Sprintf("1 error occurred:\n\t* %s\n\n", es[0])
	}

	var b strings.Builder
	// Pre-allocate assuming ~30 chars per error message on average
	b.Grow(32 + len(es)*30)
	fmt.Fprintf(&b, "%d errors occurred:\n", len(es))
	for _, err := range es {
		b.WriteString("\t* ")
		b.WriteString(err.Error())
		b.WriteString("\n")
	}
	b.WriteString("\n")

	return b.String()
}
