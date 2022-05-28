package prettier

func Wrap(left, doc, right, line Doc) Doc {
	return Group{
		Doc: Concat{
			left,
			Indent{
				Doc: Concat{
					line,
					doc,
				},
			},
			line,
			right,
		},
	}
}

func WrapBrackets(doc, line Doc) Doc {
	return Wrap(Text("["), doc, Text("]"), line)
}

func WrapParentheses(doc, line Doc) Doc {
	return Wrap(Text("("), doc, Text(")"), line)
}

func WrapBraces(doc, line Doc) Doc {
	return Wrap(Text("{"), doc, Text("}"), line)
}

// Join returns a document where the given documents
// are separated with the specified separator document
//
func Join(sep Doc, docs ...Doc) Doc {
	switch len(docs) {
	case 0:
		return nil

	case 1:
		return docs[0]

	default:
		result := make([]Doc, 0, len(docs))

		for i, doc := range docs {
			if i > 0 {
				result = append(result, sep)
			}

			result = append(result, doc)
		}

		return Concat(result)
	}
}
