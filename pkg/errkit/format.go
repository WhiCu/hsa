package errkit

import (
	"strconv"
	"strings"
)

type ErrorFormatFunc func([]error) string

// ⚡ Bolt: Using strings.Builder instead of slice allocation + strings.Join to reduce memory allocations and improve performance
func ListFormatFunc(es []error) string {
	if len(es) == 1 {
		// ⚡ Bolt: Replace fmt.Sprintf with string concatenation to improve performance on hot path
		return "1 error occurred:\n\t* " + es[0].Error() + "\n\n"
	}

	var b strings.Builder
	// Pre-allocate assuming ~30 chars per error message on average
	b.Grow(32 + len(es)*30)
	// ⚡ Bolt: Replace fmt.Fprintf with strconv.Itoa and string concatenation to avoid reflection overhead on hot path
	b.WriteString(strconv.Itoa(len(es)))
	b.WriteString(" errors occurred:\n")
	for _, err := range es {
		b.WriteString("\t* ")
		b.WriteString(err.Error())
		b.WriteString("\n")
	}
	b.WriteString("\n")

	return b.String()
}
