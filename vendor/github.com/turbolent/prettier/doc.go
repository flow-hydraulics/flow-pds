package prettier

type Doc interface {
	isDoc()
	Flatten() Doc
}

// Text is text which does not contain newline characters
//
type Text string

var _ Doc = Text("")

func (Text) isDoc() {}

func (t Text) Flatten() Doc {
	return t
}

var Space = Text(" ")

// Line is a line break.
// When flattened it is replaced with a space
//
type Line struct{}

var _ Doc = Line{}

func (Line) isDoc() {}

func (l Line) Flatten() Doc {
	return Space
}

// SoftLine is a line break.
// When flattened it is replaced with nothing
//
type SoftLine struct{}

var _ Doc = SoftLine{}

func (SoftLine) isDoc() {}

func (l SoftLine) Flatten() Doc {
	return nil
}

// HardLine is a line break.
// When flattened it is not replaced
//
type HardLine struct{}

var _ Doc = HardLine{}

func (HardLine) isDoc() {}

func (l HardLine) Flatten() Doc {
	return l
}

// Indent increases the level of indentation
// for the nested document
//
type Indent struct {
	Doc Doc
}

var _ Doc = Indent{}

func (Indent) isDoc() {}

func (i Indent) Flatten() Doc {
	return Indent{
		Doc: i.Doc.Flatten(),
	}
}

// Concat combines multiple documents
//
type Concat []Doc

var _ Doc = Concat(nil)

func (Concat) isDoc() {}

func (c Concat) Flatten() Doc {
	result := make([]Doc, len(c))
	for i, doc := range c {
		result[i] = doc.Flatten()
	}
	return Concat(result)
}

// Group marks a document to be flattened
// if it does not fit on one line
//
type Group struct {
	Doc Doc
}

var _ Doc = Group{}

func (Group) isDoc() {}

func (g Group) Flatten() Doc {
	return g.Doc.Flatten()
}
