package pipeline

import "hermes-decompiler/pkg/database/models"

// Fast, hash-set based equivalents of the matcher hot loops. The originals are O(|app|*|pkg|)
// nested loops over hex strings; these are O(|app|+|pkg|).

type tripleKey struct{ s, c1, c2 string }

func keyOf(h *models.FunctionObjectHashModel) tripleKey {
	return tripleKey{h.Structural.Hash, h.ContentAR1.Hash, h.ContentAR2.Hash}
}

// AppHashSet builds the exact-3-AR lookup set for an app bundle (build once, reuse per package).
func AppHashSet(app []*models.FunctionObjectHashModel) map[tripleKey]struct{} {
	m := make(map[tripleKey]struct{}, len(app))
	for _, h := range app {
		m[keyOf(h)] = struct{}{}
	}
	return m
}

// CompareAppWithPackageFast returns the same CombinedMatches/TotalMatchPercentage as
// CompareAppWithPackage (count of package hashes whose S+C1+C2 triple occurs in the app).
func CompareAppWithPackageFast(appSet map[tripleKey]struct{}, nApp int, pkg []*models.FunctionObjectHashModel) *MatchResults {
	r := &MatchResults{TotalAppHashes: nApp, TotalPackageHashes: len(pkg)}
	r.TotalC1Hashes, r.TotalC2Hashes = CountPackageMetadata(pkg)
	for _, h := range pkg {
		if _, ok := appSet[keyOf(h)]; ok {
			r.CombinedMatches++
		}
	}
	r.TotalAmountOfMatches = r.CombinedMatches
	if len(pkg) > 0 {
		r.TotalMatchPercentage = float64(r.CombinedMatches) / float64(len(pkg)) * 100
	}
	return r
}

// FindRNBaselineCandidatesFast picks the baseline (same HBC version) with the most exact
// triple matches and removes those functions from the app hashes.
// Difference vs FindRNBaselineCandidates: the original counts a "Combined" match only if
// the first baseline entry hit in its scan order is a full match; here any full match counts.
// Ties are broken by RN version string for determinism.
func FindRNBaselineCandidatesFast(hashes []*models.FunctionObjectHashModel, hbcVersion string) ([]*models.FunctionObjectHashModel, string) {
	best, bestN := "", -1
	sets := map[string]map[tripleKey]struct{}{}
	for rn, b := range rnBaselines {
		if b.HermesBytecodeVersion != hbcVersion {
			continue
		}
		s := make(map[tripleKey]struct{}, len(b.Hashes))
		for _, h := range b.Hashes {
			s[tripleKey{h.StructuralHash, h.ContentIR1Hash, h.ContentIR2Hash}] = struct{}{}
		}
		sets[rn] = s
		n := 0
		for _, h := range hashes {
			if _, ok := s[keyOf(h)]; ok {
				n++
			}
		}
		if n > bestN || (n == bestN && rn < best) {
			best, bestN = rn, n
		}
	}
	if best == "" {
		return hashes, ""
	}
	out := make([]*models.FunctionObjectHashModel, 0, len(hashes))
	for _, h := range hashes {
		if _, ok := sets[best][keyOf(h)]; !ok {
			out = append(out, h)
		}
	}
	return out, best
}

// CheckForDuplicatesFast is the O(n) version of CheckForDuplicates (unique C2 hashes, first-seen order).
func CheckForDuplicatesFast(hashes []*models.FunctionObjectHashModel) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, h := range hashes {
		if _, ok := seen[h.ContentAR2.Hash]; !ok {
			seen[h.ContentAR2.Hash] = struct{}{}
			out = append(out, h.ContentAR2.Hash)
		}
	}
	return out
}
