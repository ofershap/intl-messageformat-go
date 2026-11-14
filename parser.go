package messageformat

import (
	"fmt"
	"strconv"
	"strings"
)

// AST for ICU MessageFormat 1 (FormatJS dialect, tags enabled).
type (
	literalNode string
	poundNode   struct{}
	argNode     struct{ name string }
	numberNode  struct {
		name  string
		style string // "", "integer", "percent", "currency" or a custom format name
	}
	dateTimeNode struct {
		name   string
		isTime bool
		style  string // full/long/medium/short, or raw style token (typed errors at format)
	}
	selectNode struct {
		name    string
		options []selectOption
	}
	selectOption struct {
		key   string
		nodes []node
	}
	pluralNode struct {
		name    string
		ordinal bool
		offset  float64
		exact   []exactOption // =N selectors, matched on the raw value
		options []selectOption
	}
	exactOption struct {
		value float64
		nodes []node
	}
	tagNode struct {
		name     string
		children []node
	}
	node interface{}
)

type parser struct {
	src []rune
	pos int
}

func parseMessage(src string) ([]node, error) {
	p := &parser{src: []rune(src)}
	nodes, err := p.parseNodes(false, 0)
	if err != nil {
		return nil, err
	}
	if p.pos < len(p.src) {
		return nil, fmt.Errorf("messageformat: unbalanced message: unexpected %q at offset %d", string(p.src[p.pos]), p.pos)
	}
	return nodes, nil
}

// parseNodes reads until end or the closing brace/tag that owns this level.
// closing: 0 = top level, '}' = inside braces, '<' = inside a tag.
func (p *parser) parseNodes(inPlural bool, closing rune) ([]node, error) {
	var nodes []node
	var lit strings.Builder
	flush := func() {
		if lit.Len() > 0 {
			nodes = append(nodes, literalNode(lit.String()))
			lit.Reset()
		}
	}
	for p.pos < len(p.src) {
		r := p.src[p.pos]
		switch {
		case r == '}':
			if closing == '}' {
				flush()
				return nodes, nil
			}
			return nil, fmt.Errorf("messageformat: unbalanced message: unexpected '}' at offset %d", p.pos)
		case r == '<' && closing == '<' && p.peekTagClose():
			flush()
			return nodes, nil
		case r == '{':
			flush()
			n, err := p.parseArg()
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, n)
		case r == '#':
			if inPlural {
				flush()
				nodes = append(nodes, poundNode{})
				p.pos++
			} else {
				lit.WriteRune(r)
				p.pos++
			}
		case r == '<':
			if n, ok, err := p.parseTag(); err != nil {
				return nil, err
			} else if ok {
				flush()
				nodes = append(nodes, n)
			} else {
				lit.WriteRune(r)
				p.pos++
			}
		case r == '\'':
			// FormatJS quoting: an apostrophe quotes only before { } # |;
			// '' is a literal apostrophe; otherwise the apostrophe is literal.
			if p.pos+1 < len(p.src) && p.src[p.pos+1] == '\'' {
				lit.WriteByte('\'')
				p.pos += 2
			} else if p.pos+1 < len(p.src) && isQuotable(p.src[p.pos+1]) {
				p.pos++
				for p.pos < len(p.src) {
					if p.src[p.pos] == '\'' {
						if p.pos+1 < len(p.src) && p.src[p.pos+1] == '\'' {
							lit.WriteByte('\'')
							p.pos += 2
							continue
						}
						p.pos++
						break
					}
					lit.WriteRune(p.src[p.pos])
					p.pos++
				}
			} else {
				lit.WriteByte('\'')
				p.pos++
			}
		default:
			lit.WriteRune(r)
			p.pos++
		}
	}
	if closing != 0 {
		return nil, fmt.Errorf("messageformat: unbalanced message: missing closing %q", string(closing))
	}
	flush()
	return nodes, nil
}

func isQuotable(r rune) bool { return r == '{' || r == '}' || r == '#' || r == '|' }

func (p *parser) parseArg() (node, error) {
	start := p.pos
	p.pos++ // consume '{'
	p.skipSpaces()
	name := p.readUntil(func(r rune) bool { return r == ',' || r == '}' })
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("messageformat: empty argument name at offset %d", start)
	}
	if p.pos >= len(p.src) {
		return nil, fmt.Errorf("messageformat: unterminated argument at offset %d", start)
	}
	if p.src[p.pos] == '}' {
		p.pos++
		return argNode{name}, nil
	}
	p.pos++ // consume ','
	p.skipSpaces()
	typ := p.readUntil(func(r rune) bool { return r == ',' || r == '}' })
	typ = strings.TrimSpace(strings.ToLower(typ))
	switch typ {
	case "number", "date", "time":
		if p.pos >= len(p.src) || p.src[p.pos] != ',' {
			if p.pos < len(p.src) && p.src[p.pos] == '}' {
				p.pos++
				if typ == "number" {
					return numberNode{name: name}, nil
				}
				return dateTimeNode{name: name, isTime: typ == "time"}, nil
			}
			return nil, fmt.Errorf("messageformat: unterminated argument at offset %d", start)
		}
		p.pos++
		style := strings.TrimSpace(p.readUntil(func(r rune) bool { return r == '}' }))
		if p.pos >= len(p.src) {
			return nil, fmt.Errorf("messageformat: unterminated argument at offset %d", start)
		}
		p.pos++
		if typ == "number" {
			return numberNode{name: name, style: style}, nil
		}
		return dateTimeNode{name: name, isTime: typ == "time", style: style}, nil
	case "select", "plural", "selectordinal":
		if p.pos >= len(p.src) || p.src[p.pos] != ',' {
			return nil, fmt.Errorf("messageformat: %s argument %q requires options at offset %d", typ, name, start)
		}
		p.pos++
		if typ == "select" {
			opts, err := p.parseOptions(false)
			if err != nil {
				return nil, err
			}
			return selectNode{name: name, options: opts}, nil
		}
		// plural / selectordinal: optional offset, then options with =N keys
		p.skipSpaces()
		offset := 0.0
		if p.matchKeyword("offset") {
			p.skipSpaces()
			if p.pos >= len(p.src) || p.src[p.pos] != ':' {
				return nil, fmt.Errorf("messageformat: malformed plural offset at offset %d", p.pos)
			}
			p.pos++
			p.skipSpaces()
			num := p.readUntil(func(r rune) bool { return r == ' ' || r == '\t' || r == '\n' || r == '{' })
			v, err := strconv.ParseFloat(strings.TrimSpace(num), 64)
			if err != nil {
				return nil, fmt.Errorf("messageformat: malformed plural offset %q", num)
			}
			offset = v
		}
		opts, exact, err := p.parsePluralOptions()
		if err != nil {
			return nil, err
		}
		return pluralNode{name: name, ordinal: typ == "selectordinal", offset: offset, options: opts, exact: exact}, nil
	default:
		return nil, fmt.Errorf("messageformat: unsupported argument type %q at offset %d", typ, start)
	}
}

// parseOptions parses `key {message}` pairs for select; requires "other".
func (p *parser) parseOptions(inPlural bool) ([]selectOption, error) {
	opts, _, err := p.parseKeyedOptionsFull(inPlural)
	return opts, err
}

func (p *parser) parsePluralOptions() ([]selectOption, []exactOption, error) {
	return p.parseKeyedOptionsFull(true)
}

func (p *parser) parseKeyedOptionsFull(inPlural bool) ([]selectOption, []exactOption, error) {
	var opts []selectOption
	var exact []exactOption
	for {
		p.skipSpaces()
		if p.pos >= len(p.src) {
			return nil, nil, fmt.Errorf("messageformat: unbalanced message: missing '}' for options")
		}
		if p.src[p.pos] == '}' {
			p.pos++
			break
		}
		key := p.readUntil(func(r rune) bool { return r == ' ' || r == '\t' || r == '\n' || r == '{' })
		p.skipSpaces()
		if p.pos >= len(p.src) || p.src[p.pos] != '{' {
			return nil, nil, fmt.Errorf("messageformat: expected '{' after option key %q", key)
		}
		p.pos++
		body, err := p.parseNodes(inPlural, '}')
		if err != nil {
			return nil, nil, err
		}
		p.pos++ // consume '}'
		if strings.HasPrefix(key, "=") {
			v, err := strconv.ParseFloat(key[1:], 64)
			if err != nil {
				return nil, nil, fmt.Errorf("messageformat: malformed exact selector %q", key)
			}
			exact = append(exact, exactOption{value: v, nodes: body})
		} else {
			opts = append(opts, selectOption{key: key, nodes: body})
		}
	}
	hasOther := false
	for _, o := range opts {
		if o.key == "other" {
			hasOther = true
		}
	}
	if !hasOther {
		return nil, nil, fmt.Errorf("messageformat: missing required 'other' option")
	}
	return opts, exact, nil
}

// parseTag parses <name>children</name> or <name/>. ok=false when the text
// after '<' is not tag syntax, in which case '<' should be literal.
func (p *parser) parseTag() (node, bool, error) {
	save := p.pos
	p.pos++ // consume '<'
	name := p.readWhile(func(r rune) bool {
		return r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
	})
	if name == "" || !((name[0] >= 'a' && name[0] <= 'z') || (name[0] >= 'A' && name[0] <= 'Z')) {
		p.pos = save
		return nil, false, nil
	}
	if p.pos < len(p.src) && p.src[p.pos] == '/' && p.pos+1 < len(p.src) && p.src[p.pos+1] == '>' {
		p.pos += 2
		return tagNode{name: name}, true, nil
	}
	if p.pos >= len(p.src) || p.src[p.pos] != '>' {
		p.pos = save
		return nil, false, nil
	}
	p.pos++
	children, err := p.parseNodes(false, '<')
	if err != nil {
		return nil, true, err
	}
	// expect '</name>'
	if !p.matchTagClose(name) {
		return nil, true, fmt.Errorf("messageformat: unbalanced tag <%s>", name)
	}
	return tagNode{name: name, children: children}, true, nil
}

// peekTagClose reports whether the current position starts a closing tag
// (used to end a tag body).
func (p *parser) peekTagClose() bool {
	return p.pos+1 < len(p.src) && p.src[p.pos] == '<' && p.src[p.pos+1] == '/'
}

func (p *parser) matchTagClose(name string) bool {
	want := "</" + name + ">"
	if p.pos+len([]rune(want)) > len(p.src) {
		return false
	}
	for i, r := range []rune(want) {
		if p.src[p.pos+i] != r {
			return false
		}
	}
	p.pos += len([]rune(want))
	return true
}

func (p *parser) skipSpaces() {
	for p.pos < len(p.src) && (p.src[p.pos] == ' ' || p.src[p.pos] == '\t' || p.src[p.pos] == '\n' || p.src[p.pos] == '\r') {
		p.pos++
	}
}

func (p *parser) matchKeyword(kw string) bool {
	rs := []rune(kw)
	if p.pos+len(rs) > len(p.src) {
		return false
	}
	for i, r := range rs {
		if p.src[p.pos+i] != r {
			return false
		}
	}
	// keyword must not be followed by identifier chars
	if p.pos+len(rs) < len(p.src) {
		n := p.src[p.pos+len(rs)]
		if (n >= 'a' && n <= 'z') || (n >= 'A' && n <= 'Z') || (n >= '0' && n <= '9') || n == '_' {
			return false
		}
	}
	p.pos += len(rs)
	return true
}

func (p *parser) readUntil(stop func(rune) bool) string {
	start := p.pos
	for p.pos < len(p.src) && !stop(p.src[p.pos]) {
		p.pos++
	}
	return string(p.src[start:p.pos])
}

func (p *parser) readWhile(ok func(rune) bool) string {
	start := p.pos
	for p.pos < len(p.src) && ok(p.src[p.pos]) {
		p.pos++
	}
	return string(p.src[start:p.pos])
}
