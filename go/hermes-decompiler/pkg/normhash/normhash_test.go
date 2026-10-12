package normhash

import "testing"

func TestNSFoldsWidth(t *testing.T) {
	a := NS("pc=1|GetByIdShort|LoadConstUInt8|JmpTrueLong|Ret|")
	b := NS("pc=1|GetById|LoadConstInt|JmpTrue|Ret|")
	if a != b {
		t.Fatalf("%q != %q", a, b)
	}
}

func TestNCMergesAndDropsShort(t *testing.T) {
	if NC("fulfilled|x", "then") != NC("", "then|fulfilled|ab") {
		t.Fatal("C1/C2 flip or short names not normalized")
	}
}
