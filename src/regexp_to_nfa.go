package main

import (
	"sort"
	"unicode/utf8"
)

// nfaTransfer is a conditional transition between NFA states.
type nfaTransfer struct {
	target  *nfaState
	isEmpty bool
	lower   []rune
	upper   []rune
}

// copy returns a copy of NFA transfer.
func (t *nfaTransfer) copy() *nfaTransfer {
	s := &nfaTransfer{
		target:  t.target,
		isEmpty: t.isEmpty,
		lower:   make([]rune, len(t.lower)),
		upper:   make([]rune, len(t.upper))}
	copy(s.lower, t.lower)
	copy(s.upper, t.upper)
	return s
}

// nfaState is a state in an NFA.
type nfaState struct {
	transfers []*nfaTransfer
}

// peek returns last transfer in the state.
func (s *nfaState) peek() *nfaTransfer {
	return s.transfers[len(s.transfers)-1]
}

// nfa is an nondeterministic finite automaton. Note that start state is always
// the first state (whose index is 0), while the end state is always the last state.
type nfa struct {
	toStart []*nfaTransfer
	toEnd   []*nfaTransfer
	states  []*nfaState
}

// startState returns the start state of the NFA.
func (n *nfa) startState() *nfaState { return n.states[0] }

// endState returns the end state of the NFA.
func (n *nfa) endState() *nfaState { return n.states[len(n.states)-1] }

// concatenate connects two NFAs together in series (n in the front).
func (n *nfa) concatenate(f *nfa) {
	nEnd := n.endState()
	if len(nEnd.transfers) == 0 || len(f.toStart) == 0 {
		// merge n's end state and f's start state
		nEnd.transfers = append(nEnd.transfers, f.startState().transfers...)
		n.states = append(n.states, f.states[1:]...)
		for _, t := range f.toStart {
			t.target = nEnd
		}
	} else {
		// nEnd -> fStart
		nEnd.transfers = append(nEnd.transfers, &nfaTransfer{f.startState(), true, nil, nil})
		n.states = append(n.states, f.states...)
	}
	n.toEnd = f.toEnd
}

// alternate connects two NFAs together in parallel.
func (n *nfa) alternate(f *nfa) {
	var newStates []*nfaState
	// process start states of n and f
	nStart, fStart := n.startState(), f.startState()
	if len(n.toStart) == 0 && len(f.toStart) == 0 {
		// merge start states of n and f
		nStart.transfers = append(nStart.transfers, fStart.transfers...)
		newStates = append(newStates, nStart)
	} else if len(n.toStart) == 0 {
		// nStart -> fStart
		nStart.transfers = append(nStart.transfers, &nfaTransfer{fStart, true, nil, nil})
		newStates = append(newStates, nStart, fStart)
	} else if len(f.toStart) == 0 {
		// fStart -> nStart
		fStart.transfers = append(fStart.transfers, &nfaTransfer{nStart, true, nil, nil})
		newStates = append(newStates, fStart, nStart)
	} else {
		// add new start state
		start := &nfaState{}
		start.transfers = append(start.transfers, &nfaTransfer{nStart, true, nil, nil})
		start.transfers = append(start.transfers, &nfaTransfer{fStart, true, nil, nil})
		newStates = append(newStates, start, nStart, fStart)
	}
	n.toStart = nil

	// process inner states of n and f
	newStates = append(newStates, n.states[1:len(n.states)-1]...)
	newStates = append(newStates, f.states[1:len(f.states)-1]...)

	// process end states of n and f
	nEnd, fEnd := n.endState(), f.endState()
	if len(nEnd.transfers) == 0 && len(fEnd.transfers) == 0 {
		// merge end states of n and f
		newStates = append(newStates, nEnd)
		for _, t := range f.toEnd {
			t.target = nEnd
		}
		n.toEnd = append(n.toEnd, f.toEnd...)
	} else if len(nEnd.transfers) == 0 {
		// fEnd -> nEnd
		fEnd.transfers = append(fEnd.transfers, &nfaTransfer{nEnd, true, nil, nil})
		newStates = append(newStates, fEnd, nEnd)
		n.toEnd = append(n.toEnd, fEnd.peek())
	} else if len(fEnd.transfers) == 0 {
		// nEnd -> fEnd
		nEnd.transfers = append(nEnd.transfers, &nfaTransfer{fEnd, true, nil, nil})
		newStates = append(newStates, nEnd, fEnd)
		f.toEnd = append(f.toEnd, nEnd.peek())
		n.toEnd = f.toEnd
	} else {
		// add new end state
		end := &nfaState{}
		nEnd.transfers = append(nEnd.transfers, &nfaTransfer{end, true, nil, nil})
		fEnd.transfers = append(fEnd.transfers, &nfaTransfer{end, true, nil, nil})
		newStates = append(newStates, nEnd, fEnd, end)
		n.toEnd = []*nfaTransfer{nEnd.peek(), fEnd.peek()}
	}

	// done
	n.states = newStates
}

// repeatZeroTimesAndMore repeats the NFA for zero times and more.
func (n *nfa) repeatZeroTimesAndMore() {
	start, end := n.startState(), n.endState()
	start.transfers = append(start.transfers, &nfaTransfer{end, true, nil, nil})
	end.transfers = append(end.transfers, &nfaTransfer{start, true, nil, nil})
	n.toStart = append(n.toStart, end.peek())
	n.toEnd = append(n.toEnd, start.peek())
}

// repeatOnceAndMore repeats the NFA for once and more.
func (n *nfa) repeatOnceAndMore() {
	start, end := n.startState(), n.endState()
	end.transfers = append(end.transfers, &nfaTransfer{start, true, nil, nil})
	n.toStart = append(n.toStart, end.transfers[len(end.transfers)-1])
}

// repeatOnceAndLess repeats this NFA for once and less.
func (n *nfa) repeatOnceAndLess() {
	start, end := n.startState(), n.endState()
	start.transfers = append(start.transfers, &nfaTransfer{end, true, nil, nil})
	n.toEnd = append(n.toEnd, start.peek())
}

// nfaHelper helps parse regular expression.
type nfaHelper struct {
	par, alt *nfa
	stack    []*nfa
}

// peek returns the last NFA in the stack.
func (h *nfaHelper) peek() *nfa {
	if len(h.stack) == 0 {
		return nil
	}
	return h.stack[len(h.stack)-1]
}

// pop returns the last NFA in the stack and removes it from the stack.
func (h *nfaHelper) pop() *nfa {
	t := h.peek()
	if t != nil {
		h.stack = h.stack[:len(h.stack)-1]
	}
	return t
}

// char adds a new NFA into stack.
func (h *nfaHelper) char(lower, upper []rune) {
	start, end := &nfaState{}, &nfaState{}
	start.transfers = append(start.transfers, &nfaTransfer{end, false, lower, upper})
	h.stack = append(h.stack, &nfa{nil, []*nfaTransfer{start.peek()}, []*nfaState{start, end}})
}

// repeat repeats the last NFA in the stack.
func (h *nfaHelper) repeat(r rune) {
	if r == '*' {
		h.stack[len(h.stack)-1].repeatZeroTimesAndMore()
	} else if r == '+' {
		h.stack[len(h.stack)-1].repeatOnceAndMore()
	} else if r == '?' {
		h.stack[len(h.stack)-1].repeatOnceAndLess()
	}
}

// alter pushes an alternative mark into stack.
func (h *nfaHelper) alter() {
	h.stack = append(h.stack, h.alt)
}

// parenthesis pushes an parenthesis mark into stack.
func (h *nfaHelper) parenthesis() {
	h.stack = append(h.stack, h.par)
}

// group pops NFAs from stack until there is a parenthesis mark or alternative mark,
// or the stack is empty, and connects these NFA into a single one.
func (h *nfaHelper) group() {
	// concatenate all NFAs after alternative mark (if exists)
	var alters []*nfa
	for {
		last := len(h.stack) - 1
		for last >= 0 && h.stack[last] != h.par && h.stack[last] != h.alt {
			last--
		}

		if last == len(h.stack)-1 {
			if last == -1 || h.stack[last] == h.par {
				panic("empty group")
			}
			if h.stack[last] == h.alt {
				panic("empty alternative")
			}
		}

		n := h.stack[last+1]
		for i := last + 2; i < len(h.stack); i++ {
			n.concatenate(h.stack[i])
		}
		alters = append(alters, n)
		h.stack = h.stack[:last+1]

		if t := h.pop(); t == nil || t == h.par {
			break
		}
	}

	// merge all alternatives after the last parenthesis mark
	n := alters[len(alters)-1]
	for i := len(alters) - 2; i >= 0; i-- {
		n.alternate(alters[i])
	}
	h.stack = append(h.stack, n)
}

// constructNFA receives regular expression and outputs NFA.
func constructNFA(regexp string) *nfa {
	var parCnt int
	re := []rune(regexp)
	p := nfaHelper{&nfa{}, &nfa{}, []*nfa{}}
	for i := 0; i < len(re); i++ {
		switch re[i] {
		case '(':
			parCnt++
			p.parenthesis()
		case ')':
			if p.peek() == nil || p.peek() == p.par || p.peek() == p.alt {
				panic("invalid parenthesis")
			}
			if parCnt == 0 {
				panic("mismatched parentheses")
			}
			parCnt--
			p.group()
		case '*', '+', '?':
			if p.peek() == nil || p.peek() == p.par || p.peek() == p.alt ||
				i == 0 || re[i-1] == '*' || re[i-1] == '+' || re[i-1] == '?' {
				panic("invalid repeat")
			}
			p.repeat(re[i])
		case '|':
			if p.peek() == nil || p.peek() == p.par || p.peek() == p.alt {
				panic("invalid alternative")
			}
			p.alter()
		case '.':
			p.char([]rune{0, '\n' + 1}, []rune{'\n' - 1, utf8.MaxRune})
		case '[':
			i++
			// negative flag
			neg := false
			if re[i] == '^' {
				neg = true
				i++
			}

			// read a character from regular expression
			getChar := func() (result rune) {
				if re[i] == '\\' {
					i++
					if re[i] == '^' || re[i] == '-' {
						result = re[i]
					} else {
						result = decodeEscapable(re[i])
					}
				} else {
					result = re[i]
				}
				i++
				return result
			}

			// collect characters in the class
			var area [][]rune
			for {
				if re[i] == ']' {
					break
				}
				ch1 := getChar()
				if re[i] == '-' {
					i++
					ch2 := getChar()
					if ch1 > ch2 {
						panic("illegal range")
					}
					area = append(area, []rune{ch1, ch2})
				} else {
					area = append(area, []rune{ch1, ch1})
				}
			}

			// construct new NFA
			lower, upper := sortCharacterClass(neg, area)
			p.char(lower, upper)
		case '\\':
			i++
			r := decodeEscapable(re[i])
			p.char([]rune{r}, []rune{r})
		default:
			p.char([]rune{re[i]}, []rune{re[i]})
		}
	}

	if parCnt != 0 {
		panic("mismatched parentheses")
	}
	p.group()
	return p.pop()
}

// sortCharacterClass processes raw character class and outputs .
func sortCharacterClass(neg bool, area [][]rune) ([]rune, []rune) {
	if len(area) == 0 {
		panic("empty character class")
	}

	// sort and remove overlap
	sort.Slice(area, func(i, j int) bool {
		return area[i][0] == area[j][0] && area[i][1] > area[j][1] || area[i][0] < area[j][0]
	})
	var newArea [][]rune
	for _, p := range area {
		if len(newArea) == 0 || newArea[len(newArea)-1][1] < p[0]-1 {
			newArea = append(newArea, p)
		} else if newArea[len(newArea)-1][1] < p[1] {
			newArea[len(newArea)-1][1] = p[1]
		}
	}
	area = newArea

	// if neg, invert the area
	if neg {
		var last rune
		var invArea [][]rune
		// note that area[0][0] cannot be 0 because it's unable to input
		for i := 0; i < len(area); i++ {
			invArea = append(invArea, []rune{last, area[i][0] - 1})
			last = area[i][1] + 1
		}
		if last <= utf8.MaxRune {
			invArea = append(invArea, []rune{last, utf8.MaxRune})
		}
		area = invArea
	}

	// convert to required format
	lower := make([]rune, len(area))
	upper := make([]rune, len(area))
	for i, p := range area {
		lower[i] = p[0]
		upper[i] = p[1]
	}
	return lower, upper
}

// decodeEscapable returns real value of escaped character.
func decodeEscapable(r rune) rune {
	escape := map[rune]rune{
		'\\': '\\', '(': '(', ')': ')', '*': '*', '+': '+', '?': '?',
		'|': '|', '.': '.', '[': '[', ']': ']',
		't': '\t', 'r': '\r', 'n': '\n',
	}
	if v, ok := escape[r]; ok {
		return v
	}
	panic("inescapable character")
}
