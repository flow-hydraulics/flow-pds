package prettier

// simpleDoc is a concatenation of items,
// where each item is either a text
// or a line break indented a given amount
//
type simpleDoc interface {
	isSimpleDoc()
}

// simpleText is a concatenation of some text
// and a (lazily, and cached) next item
//
type simpleText struct {
	text string
	next *simpleDocCache
}

var _ simpleDoc = simpleText{}

func (simpleText) isSimpleDoc() {}

// simpleLine is line break indented a given amount
// and a (lazily, and cached) next item
//
type simpleLine struct {
	indent int
	next   *simpleDocCache
}

var _ simpleDoc = simpleLine{}

func (simpleLine) isSimpleDoc() {}

// simpleDocCache is a lazily computed
// and cached simple document.
//
// Construct with the getter.
// Use get to compute and cache the document
//
type simpleDocCache struct {
	doc    simpleDoc
	getter func() simpleDoc
}

func (c *simpleDocCache) get() simpleDoc {
	// If the getter is still set,
	// then use it to compute the document,
	// cache the result, and indicate the
	// computation is done
	if c.getter != nil {
		c.doc = c.getter()
		c.getter = nil
	}
	return c.doc
}
