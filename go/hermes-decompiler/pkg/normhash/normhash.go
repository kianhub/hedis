// Package normhash computes bundle-context-robust function fingerprints from the thesis IRs.
//
// Diagnosis (docs/eval/PROGRESS.md M6): the same package function compiled standalone vs inside an
// app differs in (1) operand-width opcode variants (GetByIdShort/GetById, J*Long, LoadConstUInt8/Int/Zero)
// that depend on the size of the whole bundle's string table and jump distances, and (2) strings moving
// between C1 and C2 because the "identifier" flag is bundle-wide. NS folds (1); NC merges C1 and C2
// and drops <=2-char (minifier-renamed) names.
package normhash

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"sort"
	"strings"
)

var width = regexp.MustCompile(`(Short|LongIndex|Long)$`)
var num = map[string]bool{"LoadConstUInt8": true, "LoadConstInt": true, "LoadConstZero": true, "LoadConstDouble": true}

// NS normalizes the structural IR ("pc=N|Op|Op|...").
func NS(s string) string {
	t := strings.Split(s, "|")
	out := make([]string, 0, len(t))
	for i, o := range t {
		if i == 0 {
			out = append(out, o)
			continue
		}
		if o == "" {
			continue
		}
		o = width.ReplaceAllString(o, "")
		if num[o] {
			o = "LoadConstNum"
		}
		out = append(out, o)
	}
	return strings.Join(out, "|")
}

// NC merges both content IRs into one sorted multiset of strings longer than 2 chars.
func NC(c1, c2 string) string {
	var xs []string
	for _, c := range []string{c1, c2} {
		for _, x := range strings.Split(c, "|") {
			if len(x) > 2 {
				xs = append(xs, x)
			}
		}
	}
	sort.Strings(xs)
	return strings.Join(xs, "|")
}

func sha(s string) string { x := sha256.Sum256([]byte(s)); return hex.EncodeToString(x[:]) }

// Hashes returns (NS hash, NC hash) for raw thesis IRs. NC hash is "" when the function has no strings.
func Hashes(s, c1, c2 string) (string, string) {
	nc := NC(c1, c2)
	if nc == "" {
		return sha(NS(s)), ""
	}
	return sha(NS(s)), sha(nc)
}
