package prettier

import (
	"io"
	"strings"
)

// layout writes a simple document with the given indentation
// to the specified string writer
//
func layout(writer io.StringWriter, doc simpleDoc, indent string) {
	for doc != nil {
		switch typedDoc := doc.(type) {
		case simpleLine:
			_, _ = writer.WriteString("\n")
			_, _ = writer.WriteString(strings.Repeat(indent, typedDoc.indent))
			doc = typedDoc.next.get()

		case simpleText:
			_, _ = writer.WriteString(typedDoc.text)
			doc = typedDoc.next.get()

		default:
			break
		}
	}
}
