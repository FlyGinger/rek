package main

// dfaTransfer is a conditional transition between DFA states.
type dfaTransfer struct {
	target       int
	lower, upper rune
}

// dfaState is a state in a DFA.
type dfaState struct {
	isEnd     bool
	transfers []dfaTransfer
}

// dfa is a deterministic finite automaton.
type dfa struct {
	states []dfaState
}

// nextState returns next state according to current state and input character.
func (d *dfa) nextState(state int, input rune) int {
	area := d.states[state].transfers
	left, right := 0, len(area)
	for left < right {
		middle := (left + right) / 2
		if input < area[middle].lower {
			right = middle
		} else if area[middle].upper < input {
			left = middle + 1
		} else {
			return area[middle].target
		}
	}
	return -1
}

// constructDFA receives NFA and outputs DFA.
func constructDFA(n *nfa) *dfa {
	h := constructDFAHelper(n)
	h.addDFAState(h.closure[0])
	if h.dfa.states[0].isEnd {
		panic("empty string is accepted by this NFA")
	}

	type info struct {
		state        int
		lower, upper []rune
		target       [][]int
	}
	queue := []info{{0, h.lower[0], h.upper[0], h.target[0]}}
	isVisited := map[int]bool{0: true}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]

		for i := range p.lower {
			set := make([]bool, h.size)
			dict := map[int]bool{}
			for _, s := range p.target[i] {
				if !dict[s] {
					for k, b := range h.closure[s] {
						set[k] = set[k] || b
					}
					dict[s] = true
				}
			}
			next := h.addDFAState(set)
			h.dfa.states[p.state].transfers = append(h.dfa.states[p.state].transfers,
				dfaTransfer{next, p.lower[i], p.upper[i]})

			if !isVisited[next] {
				var lower, upper []rune
				var target [][]int
				for k := range dict {
					lower, upper, target = mergeNext(lower, upper, target, h.lower[k], h.upper[k], h.target[k])
				}
				queue = append(queue, info{next, lower, upper, target})
				isVisited[next] = true
			}
		}
	}
	return h.dfa
}

// dfaHelper helps convert NFA to DFA.
type dfaHelper struct {
	size         int
	nfaStateId   map[*nfaState]int
	closure      [][]bool
	lower, upper [][]rune
	target       [][][]int
	p, m         int
	dfsState     [][]bool
	dfaStateId   map[int][]int
	dfa          *dfa
}

// dfaStateHash returns the hash value of a DFA state.
func (h *dfaHelper) dfaStateHash(set []bool) int {
	var hash int
	for _, b := range set {
		hash *= h.p
		if b {
			hash += 1
		}
		hash %= h.m
	}
	return hash
}

// addDFAState adds a DFA state into dfaStateId and return index of the state.
func (h *dfaHelper) addDFAState(set []bool) int {
	hash := h.dfaStateHash(set)
	if list, ok := h.dfaStateId[hash]; !ok {
		h.dfaStateId[hash] = []int{len(h.dfsState)}
	} else {
		for _, i := range list {
			equal := true
			for j := 0; equal && j < len(set); j++ {
				equal = equal && set[j] == h.dfsState[i][j]
			}
			if equal {
				return i
			}
		}
		h.dfaStateId[hash] = append(list, len(h.dfsState))
	}
	h.dfsState = append(h.dfsState, set)
	h.dfa.states = append(h.dfa.states, dfaState{set[len(set)-1], nil})
	return len(h.dfsState) - 1
}

// constructDFAHelper initializes a dfaHelper.
func constructDFAHelper(n *nfa) *dfaHelper {
	h := &dfaHelper{
		size:       len(n.states),
		nfaStateId: map[*nfaState]int{},
		p:          3,
		m:          100000007,
		dfaStateId: map[int][]int{},
		dfa:        &dfa{},
	}
	// number NFA states
	for i, s := range n.states {
		h.nfaStateId[s] = i
	}
	// calculate transitive closure (E(p))
	h.closure = make([][]bool, h.size)
	for i := range h.closure {
		h.closure[i] = make([]bool, h.size)
		h.closure[i][i] = true
	}
	for i, s := range n.states {
		for _, t := range s.transfers {
			if t.isEmpty {
				h.closure[i][h.nfaStateId[t.target]] = true
			}
		}
	}
	for k := 0; k < h.size; k++ {
		for i := 0; i < h.size; i++ {
			for j := 0; j < h.size; j++ {
				h.closure[i][j] = h.closure[i][j] || h.closure[i][k] && h.closure[k][j]
			}
		}
	}
	// calculate choice
	h.lower = make([][]rune, h.size)
	h.upper = make([][]rune, h.size)
	h.target = make([][][]int, h.size)
	for i := 0; i < h.size; i++ {
		for j, b := range h.closure[i] {
			if !b {
				continue
			}
			for _, t := range n.states[j].transfers {
				if t.isEmpty {
					continue
				}
				x := h.nfaStateId[t.target]
				target := make([][]int, len(t.lower))
				for k := range target {
					target[k] = []int{x}
				}
				h.lower[i], h.upper[i], h.target[i] = mergeNext(
					h.lower[i], h.upper[i], h.target[i],
					t.lower, t.upper, target)
			}
		}
	}
	return h
}

// mergeNext merges two choice set.
func mergeNext(l1, u1 []rune, t1 [][]int, l2, u2 []rune, t2 [][]int) ([]rune, []rune, [][]int) {
	var lower, upper []rune
	var target [][]int
	addChoice := func(l, u rune, t []int) {
		lower = append(lower, l)
		upper = append(upper, u)
		target = append(target, t)
	}
	copySlice := func(x []int) []int {
		y := make([]int, len(x))
		copy(y, x)
		return y
	}

	type rec struct {
		runes []rune
		idx   int
		value rune
	}
	var todo []rec

	for i, j := 0, 0; i < len(l1) || j < len(l2); {
		if j == len(l2) || i < len(l1) && u1[i] < l2[j] {
			addChoice(l1[i], u1[i], copySlice(t1[i]))
			i++
		} else if i == len(l1) || j < len(l2) && u2[j] < l1[i] {
			addChoice(l2[j], u2[j], copySlice(t2[j]))
			j++
		} else {
			if l1[i] < l2[j] {
				addChoice(l1[i], l2[j]-1, copySlice(t1[i]))
				todo = append(todo, rec{l1, i, l1[i]})
				l1[i] = l2[j]
			} else if l2[j] < l1[i] {
				addChoice(l2[j], l1[i]-1, copySlice(t2[j]))
				todo = append(todo, rec{l2, j, l2[j]})
				l2[j] = l1[i]
			} else {
				if u1[i] < u2[j] {
					addChoice(l1[i], u1[i], append(t1[i], t2[j]...))
					todo = append(todo, rec{l2, j, l2[j]})
					l2[j] = u1[i] + 1
					i++
				} else if u2[j] < u1[i] {
					addChoice(l2[j], u2[j], append(t2[j], t1[i]...))
					todo = append(todo, rec{l1, i, l1[i]})
					l1[i] = u2[j] + 1
					j++
				} else {
					addChoice(l1[i], u1[i], append(t1[i], t2[j]...))
					i++
					j++
				}
			}
		}
	}

	for i := len(todo) - 1; i >= 0; i-- {
		t := todo[i]
		t.runes[t.idx] = t.value
	}
	return lower, upper, target
}
