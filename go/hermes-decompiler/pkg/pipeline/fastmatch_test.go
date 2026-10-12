package pipeline

import (
	"fmt"
	"hermes-decompiler/pkg/database/models"
	"math/rand"
	"testing"
)

func randHashes(r *rand.Rand, n, space int) []*models.FunctionObjectHashModel {
	out := make([]*models.FunctionObjectHashModel, n)
	for i := range out {
		h := &models.FunctionObjectHashModel{}
		h.Structural.Hash = fmt.Sprint("s", r.Intn(space))
		if r.Intn(2) == 0 {
			h.ContentAR1.Hash = fmt.Sprint("a", r.Intn(space))
		}
		if r.Intn(3) > 0 {
			h.ContentAR2.Hash = fmt.Sprint("b", r.Intn(space))
		}
		out[i] = h
	}
	return out
}

func TestCompareFastEquivalent(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for it := 0; it < 50; it++ {
		app, pkg := randHashes(r, 300, 8), randHashes(r, 120, 8)
		a := CompareAppWithPackage(app, pkg)
		b := CompareAppWithPackageFast(AppHashSet(app), len(app), pkg)
		if a.CombinedMatches != b.CombinedMatches || a.TotalMatchPercentage != b.TotalMatchPercentage {
			t.Fatalf("iter %d: legacy %d/%f fast %d/%f", it, a.CombinedMatches, a.TotalMatchPercentage, b.CombinedMatches, b.TotalMatchPercentage)
		}
	}
}

func TestCheckForDuplicatesFastEquivalent(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	h := randHashes(r, 500, 20)
	a, b := CheckForDuplicates(h), CheckForDuplicatesFast(h)
	if fmt.Sprint(a) != fmt.Sprint(b) {
		t.Fatal("mismatch")
	}
}

func BenchmarkCompareLegacy(bm *testing.B) {
	r := rand.New(rand.NewSource(3))
	app, pkg := randHashes(r, 50000, 1<<30), randHashes(r, 3000, 1<<30)
	for i := 0; i < bm.N; i++ {
		CompareAppWithPackage(app, pkg)
	}
}

func BenchmarkCompareFast(bm *testing.B) {
	r := rand.New(rand.NewSource(3))
	app, pkg := randHashes(r, 50000, 1<<30), randHashes(r, 3000, 1<<30)
	set := AppHashSet(app)
	for i := 0; i < bm.N; i++ {
		CompareAppWithPackageFast(set, len(app), pkg)
	}
}
