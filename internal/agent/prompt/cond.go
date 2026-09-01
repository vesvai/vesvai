package prompt

type branch struct {
	expr  string
	fn    func(Vars) bool
	parts []Part
}

type IfPart struct {
	base
	branches []branch
}

func If(expr string, then ...Part) *IfPart {
	c := &IfPart{}
	c.branches = append(c.branches, branch{expr: expr, parts: then})
	return c
}

func IfFn(fn func(Vars) bool, then ...Part) *IfPart {
	if fn == nil {
		panic("prompt: IfFn: condition must not be nil")
	}
	c := &IfPart{}
	c.branches = append(c.branches, branch{fn: fn, parts: then})
	return c
}

func When(expr string, parts ...Part) *IfPart {
	return If(expr, parts...)
}

func (c *IfPart) ElseIf(expr string, parts ...Part) *IfPart {
	c.branches = append(c.branches, branch{expr: expr, parts: parts})
	return c
}

func (c *IfPart) Else(parts ...Part) *IfPart {
	c.branches = append(c.branches, branch{parts: parts})
	return c
}

func (c *IfPart) resolve(rc *renderCtx) ([]Part, string, error) {
	for i, b := range c.branches {
		if b.fn != nil {
			if b.fn(rc.vars) {
				return b.parts, branchLabel(c.branches, i), nil
			}
			continue
		}
		if b.expr == "" {
			return b.parts, "else", nil
		}
		ok, err := evalBool(b.expr, rc.vars)
		if err != nil {
			return nil, "", err
		}
		if ok {
			return b.parts, branchLabel(c.branches, i), nil
		}
	}
	return nil, "none", nil
}

func branchLabel(branches []branch, i int) string {
	if i == 0 {
		return "then"
	}
	if i == len(branches)-1 && branches[i].expr == "" {
		return "else"
	}
	return "elseif"
}

func (c *IfPart) renderMarkdown(rc *renderCtx) (string, error) {
	parts, _, err := c.resolve(rc)
	if err != nil {
		return "", err
	}
	return joinText(rc, parts, "\n\n")
}

func (c *IfPart) renderXML(rc *renderCtx) (string, error) {
	parts, _, err := c.resolve(rc)
	if err != nil {
		return "", err
	}
	return joinText(rc, parts, "\n")
}

func (c *IfPart) renderJSON(rc *renderCtx) (any, error) {
	parts, taken, err := c.resolve(rc)
	if err != nil {
		return nil, err
	}
	node := map[string]any{"type": "if", "taken": taken, "parts": []any{}}
	if len(c.branches) > 0 && c.branches[0].expr != "" {
		node["condition"] = c.branches[0].expr
	}
	if len(parts) > 0 {
		nodes, err := jsonNodes(rc, parts)
		if err != nil {
			return nil, err
		}
		node["parts"] = nodes
	}
	return node, nil
}

type switchCase struct {
	expr  string
	parts []Part
}

type SwitchPart struct {
	base
	expr         string
	cases        []switchCase
	defaultParts []Part
}

func Switch(expr string) *SwitchPart {
	return &SwitchPart{expr: expr}
}

func (s *SwitchPart) Case(caseExpr string, parts ...Part) *SwitchPart {
	s.cases = append(s.cases, switchCase{expr: caseExpr, parts: parts})
	return s
}

func (s *SwitchPart) Default(parts ...Part) *SwitchPart {
	s.defaultParts = parts
	return s
}

func (s *SwitchPart) resolve(rc *renderCtx) ([]Part, string, error) {
	want, err := evalValue(s.expr, rc.vars)
	if err != nil {
		return nil, "", err
	}
	for _, c := range s.cases {
		n, err := parse(c.expr)
		if err != nil {
			return nil, "", err
		}
		if isComparison(n) {
			ok, err := evalBool(c.expr, rc.vars)
			if err != nil {
				return nil, "", err
			}
			if ok {
				return c.parts, c.expr, nil
			}
			continue
		}
		got, err := evalNode(n, rc.vars)
		if err != nil {
			return nil, "", err
		}
		if equalValues(want, got) {
			return c.parts, c.expr, nil
		}
	}
	if s.defaultParts != nil {
		return s.defaultParts, "default", nil
	}
	return nil, "none", nil
}

func (s *SwitchPart) renderMarkdown(rc *renderCtx) (string, error) {
	parts, _, err := s.resolve(rc)
	if err != nil {
		return "", err
	}
	return joinText(rc, parts, "\n\n")
}

func (s *SwitchPart) renderXML(rc *renderCtx) (string, error) {
	parts, _, err := s.resolve(rc)
	if err != nil {
		return "", err
	}
	return joinText(rc, parts, "\n")
}

func (s *SwitchPart) renderJSON(rc *renderCtx) (any, error) {
	parts, taken, err := s.resolve(rc)
	if err != nil {
		return nil, err
	}
	node := map[string]any{"type": "switch", "expr": s.expr, "taken": taken, "parts": []any{}}
	if len(parts) > 0 {
		nodes, err := jsonNodes(rc, parts)
		if err != nil {
			return nil, err
		}
		node["parts"] = nodes
	}
	return node, nil
}

func branchParts(fn func(*Prompt)) []Part {
	q := New()
	fn(q)
	return q.parts
}

func (p *Prompt) If(expr string, then func(*Prompt), elseFns ...func(*Prompt)) *Prompt {
	c := If(expr, branchParts(then)...)
	for _, fn := range elseFns {
		c.Else(branchParts(fn)...)
	}
	return p.Add(c)
}

func (p *Prompt) IfFn(fn func(Vars) bool, then func(*Prompt), elseFns ...func(*Prompt)) *Prompt {
	c := IfFn(fn, branchParts(then)...)
	for _, f := range elseFns {
		c.Else(branchParts(f)...)
	}
	return p.Add(c)
}

func (p *Prompt) When(expr string, then func(*Prompt)) *Prompt {
	return p.Add(When(expr, branchParts(then)...))
}

func (p *Prompt) Switch(expr string, fn func(*SwitchPart)) *Prompt {
	s := Switch(expr)
	if fn != nil {
		fn(s)
	}
	return p.Add(s)
}
