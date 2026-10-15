package pipeline

import (
	"reflect"
	"testing"
)

func TestChooseRNVersions(t *testing.T) {
	avail := []string{"0.79", "0.74", "0.76"}
	cases := []struct {
		rn, mode string
		want     []string
		fallback bool
	}{
		{"0.76", "nearest", []string{"0.76"}, false},
		{"0.76", "union", []string{"0.76"}, false}, // covered: never falls back
		{"0.72", "nearest", []string{"0.74"}, true},
		{"0.75", "nearest", []string{"0.76"}, true}, // tie 0.74/0.76 -> newer
		{"0.78", "nearest", []string{"0.79"}, true},
		{"0.80", "nearest", []string{"0.79"}, true},
		{"0.72", "union", []string{"0.74", "0.76", "0.79"}, true},
		{"bogus", "nearest", []string{"0.79"}, true},
	}
	for _, c := range cases {
		got, note := ChooseRNVersions(c.rn, avail, c.mode)
		if !reflect.DeepEqual(got, c.want) || (note != "") != c.fallback {
			t.Errorf("%s/%s: got %v %q, want %v fallback=%v", c.rn, c.mode, got, note, c.want, c.fallback)
		}
	}
	if got, note := ChooseRNVersions("0.72", nil, "nearest"); got != nil || note == "" {
		t.Errorf("empty DB: got %v %q", got, note)
	}
}

func TestRNNum(t *testing.T) {
	a, _ := rnNum("0.72")
	b, _ := rnNum("rn074")
	c, _ := rnNum("0.80.1")
	if !(a < b && b < c) {
		t.Errorf("order %v %v %v", a, b, c)
	}
}
