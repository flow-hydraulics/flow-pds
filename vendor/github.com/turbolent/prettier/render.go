package prettier

// fits determines if the given document
// fits into the remaining line width
//
func fits(remainingWidth int, doc simpleDoc) bool {
	if remainingWidth < 0 {
		return false
	}

	if doc, ok := doc.(simpleText); ok {
		return fits(remainingWidth-len(doc.text), doc.next.get())
	}

	return true
}

// best returns the best of all possible layouts for the given documents,
// based on the maximum width for each line (the available width),
// and the line width, the number of characters already placed
// on the current line (including indentation)
//
func best(maxLineWidth int, lineWidth int, docs *layoutDocs) simpleDoc {
	if docs == nil {
		return nil
	}

	// Ignore the empty document (nil)
	if docs.doc == nil {
		return best(maxLineWidth, lineWidth, docs.next)
	}

	switch doc := docs.doc.(type) {
	case Concat:
		newDocs := docs.next
		// Prepend the documents to the linked list,
		// i.e iterate in reverse order
		for i := len(doc) - 1; i >= 0; i-- {
			newDocs = &layoutDocs{
				indent: docs.indent,
				doc:    doc[i],
				next:   newDocs,
			}
		}
		return best(maxLineWidth, lineWidth, newDocs)

	case Indent:
		return best(
			maxLineWidth,
			lineWidth,
			&layoutDocs{
				indent: docs.indent + 1,
				doc:    doc.Doc,
				next:   docs.next,
			},
		)

	case Group:
		newDocs := &layoutDocs{
			indent: docs.indent,
			doc:    doc.Doc.Flatten(),
			next:   docs.next,
		}
		flattenedDoc := best(
			maxLineWidth,
			lineWidth,
			newDocs,
		)
		if fits(maxLineWidth-lineWidth, flattenedDoc) {
			return flattenedDoc
		}

		newDocs.doc = doc.Doc
		return best(maxLineWidth, lineWidth, newDocs)

	case Line, SoftLine, HardLine:
		return simpleLine{
			indent: docs.indent,
			next: &simpleDocCache{
				getter: func() simpleDoc {
					return best(maxLineWidth, docs.indent, docs.next)
				},
			},
		}

	case Text:
		return simpleText{
			text: string(doc),
			next: &simpleDocCache{
				getter: func() simpleDoc {
					return best(maxLineWidth, lineWidth+len(doc), docs.next)
				},
			},
		}

	default:
		return nil
	}
}

// layoutDocs is a linked list of possible document layouts
//
type layoutDocs struct {
	indent int
	doc    Doc
	next   *layoutDocs
}

// pretty returns the best of all possible layouts for the given document,
// based on the maximum width for each line
//
func pretty(maxLineWidth int, doc Doc) simpleDoc {
	// Determine the best layout for the document,
	// by starting with just this document,
	// and no indentation
	docs := &layoutDocs{indent: 0, doc: doc}
	return best(maxLineWidth, 0, docs)
}
