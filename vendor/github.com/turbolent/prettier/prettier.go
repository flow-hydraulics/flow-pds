package prettier

import (
	"io"
)

// Prettier writes the given document to the specified string writer,
// indented using the given indentation, and trying to fit the content
// so that the maximum line width is not exceeded
//
func Prettier(writer io.StringWriter, doc Doc, maxLineWidth int, indent string) {
	d := pretty(maxLineWidth, doc)
	layout(writer, d, indent)
}
