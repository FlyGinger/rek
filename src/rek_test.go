package main

import (
	"fmt"
	"regexp"
	"testing"
)

func TestConstructNFA(t *testing.T) {
	wrapper := func(re string) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println(re, r)
			}
		}()
		fmt.Println(re)
		n := constructNFA(re)
		fmt.Println(convertNFAToString(n))
	}

	cases := []string{
		"\\.(a?b|(xy)+|yz*).\\t",
		"",
		"a*b*",
		"a*|b*",
		"a*|b*|c*",
		"a*|bc",
		"ab|cd",
		"[a-z0-9-]",
		"[^a-zA-Z0-9]",
		"[^.\\\\\\\\]",
		"[\\^\\-\\]]",
	}
	for _, re := range cases {
		wrapper(re)
	}
}

func TestConstructDFA(t *testing.T) {
	wrapper := func(re string) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println(re, r)
			}
		}()
		fmt.Println(re)
		n := constructNFA(re)
		fmt.Println(convertNFAToString(n))
		d := constructDFA(n)
		fmt.Println(convertDFAToString(d))
	}

	cases := []string{
		"\\.(a?b|(xy)+|yz*).\\t",
		"a*b*c*",
		"return|result",
	}
	for _, re := range cases {
		wrapper(re)
	}
}

func TestMatch(t *testing.T) {
	input := []string{
		"aaabbb123xyz",
		"aaabbbabc",
		"0xabc",
		"bbbbbbabc",
	}
	output := []bool{
		false,
		true,
		true,
		true,
	}

	r := Compile("(a*|b*)[0-9]?[a-zA-Z]+(x?y?z?|abc)")
	for i := 0; i < len(input); i++ {
		if r.Match(input[i]) != output[i] {
			t.Errorf(input[i])
		}
	}
}

func BenchmarkCompileMatch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		r := Compile("(a*|b*)[0-9]?[a-zA-Z]+(x?y?z?|abc)")
		r.Match("aaabbb123xyz")
		r.Match("aaabbbabc")
		r.Match("0xabc")
		r.Match("bbbbbbabc")
	}
}

func BenchmarkMatch(b *testing.B) {
	r := Compile("(a*|b*)[0-9]?[a-zA-Z]+(x?y?z?|abc)")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match("aaabbb123xyz")
		r.Match("aaabbbabc")
		r.Match("0xabc")
		r.Match("bbbbbbabc")
	}
}

func BenchmarkRE2CompileMatch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		r := regexp.MustCompile("(a*|b*)[0-9]?[a-zA-Z]+(x?y?z?|abc)")
		r.MatchString("aaabbb123xyz")
		r.MatchString("aaabbbabc")
		r.MatchString("0xabc")
		r.MatchString("bbbbbbabc")
	}
}

func BenchmarkRE2Match(b *testing.B) {
	r := regexp.MustCompile("(a*|b*)[0-9]?[a-zA-Z]+(x?y?z?|abc)")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.MatchString("aaabbb123xyz")
		r.MatchString("aaabbbabc")
		r.MatchString("0xabc")
		r.MatchString("bbbbbbabc")
	}
}
