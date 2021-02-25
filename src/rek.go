package main

import (
	"fmt"
	"strings"
)

type REK struct {
	d *dfa
}

func (re *REK) Match(s string) bool {
	state := 0
	for _, ch := range s {
		state = re.d.nextState(state, ch)
		if state == -1 {
			return false
		}
	}
	return re.d.states[state].isEnd
}

func Compile(re string) REK {
	n := constructNFA(re)
	//fmt.Println(convertNFAToString(n))
	d := constructDFA(n)
	//fmt.Println(convertDFAToString(d))
	return REK{d}
}

// convertNFAToString converts NFA to human-readable string.
func convertNFAToString(n *nfa) string {
	if n == nil {
		return "nil"
	}

	dict := map[*nfaState]int{}
	for i, s := range n.states {
		dict[s] = i
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("NFA with %d state(s)\n", len(n.states)))
	for i, s := range n.states {
		sb.WriteString(fmt.Sprintf("state %d\n", i))
		for _, t := range s.transfers {
			sb.WriteString(fmt.Sprintf("  -> %d", dict[t.target]))
			if t.isEmpty {
				sb.WriteString(", empty")
			} else {
				for j := range t.lower {
					sb.WriteString(fmt.Sprintf(", [%v, %v]", t.lower[j], t.upper[j]))
				}
			}
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// convertDFAToString converts DFA to human-readable string.
func convertDFAToString(d *dfa) string {
	if d == nil {
		return "nil"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("DFA with %d state(s)\n", len(d.states)))
	for i, s := range d.states {
		sb.WriteString(fmt.Sprintf("state %d", i))
		if s.isEnd {
			sb.WriteString(" (end)\n")
		} else {
			sb.WriteString("\n")
		}

		for _, t := range s.transfers {
			sb.WriteString(fmt.Sprintf("  [%v, %v] -> %d\n", t.lower, t.upper, t.target))
		}
	}
	return sb.String()
}
