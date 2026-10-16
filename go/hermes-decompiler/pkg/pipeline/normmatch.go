package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"hermes-decompiler/pkg/database/models"
	"hermes-decompiler/pkg/normhash"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NormMatchConfig: KeyMode "nsnc" (NS+NC) or "ns" (structural-only); Thresholds per package name
// with "_global" fallback (dev-tuned, eval/matchlab2.py); TieBreak selects the version by F1 of its
// keys vs the app's keys of that package instead of the highest IDF score.
type NormMatchConfig struct {
	KeyMode    string
	Thresholds map[string]float64
	TieBreak   bool
	// LowUniq: packages with <= LowUniq unique keys are scored on keys shared by <= LowDf packages
	// (react-freeze: its only unique key is a bundle artifact absent from apps).
	LowUniq, LowDf int
	// Dep: dependency implication. A package that a predicted package@version declares in its npm
	// dependencies (Deps[pkg][version]) is added when >= Dep of its keys that the parent's own keys
	// also contain are in the app (min 5 shared keys); version = most app hits, then newest.
	Dep float64
	// RootVer: choose the version on root-entry keys only (subpath rows exist only for some versions,
	// so they bias the version fraction); the package decision still uses all keys.
	RootVer bool
	Deps    map[string]map[string][]string
	// Floor (M35): drop final predictions (incl. dep-implied) whose score is below Floor (0 = off).
	Floor float64
	// Floors (M40): per-package override of Floor (e.g. react-native-gesture-handler on RN 0.85, whose app code
	// keeps only ~2-20% of its unique keys while absent apps score 0).
	Floors map[string]float64
	// VsetMargin (M35, < 0 = off): per predicted package also report the version set the bundle cannot
	// distinguish: every version whose version-selector score is >= the chosen version's score - VsetMargin,
	// plus (VsetSameKeys) every version whose set of app-matched selector keys equals the chosen version's.
	VsetMargin   float64
	VsetSameKeys bool
	// UnsupportedMinCov (M38): if > 0, refuse when the best package-level nsnc coverage
	// (|fp_keys ∩ app_keys| / |fp_keys|) across DB package versions with >= UnsupportedMinKeys
	// keys is below this floor. Distinguishes unsupported RN/bytecode (empty preds that look
	// "clean") from a real empty match. 0 = off. Default CLI: DefaultUnsupportedMinCov.
	UnsupportedMinCov float64
	// UnsupportedMinKeys: min fingerprint keys a package version must have to vote on coverage
	// (tiny packages with shared trivial shapes are ignored). 0 -> defaultUnsupportedMinKeys.
	UnsupportedMinKeys int
}

// vless compares dotted versions numerically (pre-release parts count as -1).
func vless(a, b string) bool {
	f := func(v string) []int {
		var out []int
		for _, x := range strings.FieldsFunc(v, func(r rune) bool { return r == '.' || r == '-' }) {
			n, err := strconv.Atoi(x)
			if err != nil {
				n = -1
			}
			out = append(out, n)
		}
		return out
	}
	x, y := f(a), f(b)
	for i := 0; i < len(x) && i < len(y); i++ {
		if x[i] != y[i] {
			return x[i] < y[i]
		}
	}
	return len(x) < len(y)
}

type nkey struct{ s, c string }

func (c NormMatchConfig) key(ns, nc string) nkey {
	if c.KeyMode == "ns" {
		return nkey{ns, ""}
	}
	return nkey{ns, nc}
}

type pvKeys struct {
	pkg  *models.PackageWithVulnerabilitiesModel
	keys map[nkey]struct{}
	root map[nkey]struct{} // keys from the root entry bundle (not subpath rows)
}

// DefaultUnsupportedMinCov is the M38 match-time gate for unsupported RN/bytecode.
// Tuned 2026-10-15 on real_dev + synth (m37f predicted max nsnc scores: min 0.938, median 0.998)
// vs self-compiled /workspace/rn085/app.hbc package coverages (axios 0.071, zod 0.126,
// safe-area-context 0.068; max 0.126). No zeus/v6 apps used. Mid-gap value well above the
// unsupported ceiling (~0.13) and well below the healthy floor of max package scores (~0.94).
const DefaultUnsupportedMinCov = 0.40

const defaultUnsupportedMinKeys = 20

// ErrUnsupportedBytecode is returned when overall package nsnc coverage vs the selected RN
// fingerprints is far below the healthy baseline (see DefaultUnsupportedMinCov).
var ErrUnsupportedBytecode = errors.New("React Native or bytecode version not supported")

// UnsupportedBytecodeError carries the observed max package coverage for logs/JSON status.
type UnsupportedBytecodeError struct {
	Coverage float64
	Min      float64
	BestPkg  string
}

func (e *UnsupportedBytecodeError) Error() string {
	return fmt.Sprintf("%s (max package nsnc coverage %.3f < %.3f; best %s)",
		ErrUnsupportedBytecode.Error(), e.Coverage, e.Min, e.BestPkg)
}

func (e *UnsupportedBytecodeError) Is(target error) bool {
	return target == ErrUnsupportedBytecode
}

// MaxPackageNSNCCoverage returns the highest |keys ∩ A| / |keys| among package versions with
// at least minKeys fingerprint keys, plus the best package@version name.
func MaxPackageNSNCCoverage(A map[nkey]struct{}, pvs []*pvKeys, minKeys int) (float64, string) {
	if minKeys <= 0 {
		minKeys = defaultUnsupportedMinKeys
	}
	best := 0.0
	bestName := ""
	for _, p := range pvs {
		if len(p.keys) < minKeys {
			continue
		}
		hit := 0
		for k := range p.keys {
			if _, ok := A[k]; ok {
				hit++
			}
		}
		cov := float64(hit) / float64(len(p.keys))
		if cov > best {
			best = cov
			if p.pkg != nil {
				bestName = p.pkg.PackageName + "@" + p.pkg.PackageVersion
			}
		}
	}
	return best, bestName
}

// NormMatch implements method (a)+(b)[+(c)]: normalized keys minus RN baseline, keep only keys unique
// to one package name, IDF-weighted match fraction, per-package threshold, then version selection.
// Returns one MatchResults per predicted package (TotalMatchPercentage = score*100).
func NormMatch(app []*models.FunctionObjectHashModel, rn string, cfg NormMatchConfig) ([]*MatchResults, error) {
	base := map[nkey]struct{}{}
	if b, ok := rnBaselines[rn]; ok {
		for _, h := range b.Hashes {
			base[cfg.key(h.NormS, h.NormC)] = struct{}{}
		}
	}
	var pairs [][2]string
	for _, h := range app {
		ns, nc := normhash.Hashes(h.Structural.Raw, h.ContentAR1.Raw, h.ContentAR2.Raw)
		pairs = append(pairs, [2]string{ns, nc})
	}
	return NormMatchPairs(pairs, rn, cfg, base)
}

// NormMatchPairs runs NormMatch on precomputed (NS, NC) hashes (e.g. from a --dump-app file).
func NormMatchPairs(pairs [][2]string, rn string, cfg NormMatchConfig, base map[nkey]struct{}) ([]*MatchResults, error) {
	if base == nil {
		base = map[nkey]struct{}{}
		if b, ok := rnBaselines[rn]; ok {
			for _, h := range b.Hashes {
				base[cfg.key(h.NormS, h.NormC)] = struct{}{}
			}
		}
	}
	A := map[nkey]struct{}{}
	for _, p := range pairs {
		k := cfg.key(p[0], p[1])
		if _, in := base[k]; !in {
			A[k] = struct{}{}
		}
	}
	pvs, err := loadNormKeys(rn, cfg, base)
	if err != nil {
		return nil, err
	}
	if cfg.UnsupportedMinCov > 0 {
		cov, best := MaxPackageNSNCCoverage(A, pvs, cfg.UnsupportedMinKeys)
		if cov < cfg.UnsupportedMinCov {
			return nil, &UnsupportedBytecodeError{Coverage: cov, Min: cfg.UnsupportedMinCov, BestPkg: best}
		}
	}
	byPkg := map[string][]*pvKeys{}
	df := map[nkey]map[string]struct{}{}
	for _, p := range pvs {
		n := p.pkg.PackageName
		byPkg[n] = append(byPkg[n], p)
		for k := range p.keys {
			if df[k] == nil {
				df[k] = map[string]struct{}{}
			}
			df[k][n] = struct{}{}
		}
	}
	N := float64(len(byPkg))
	dfRoot := map[nkey]map[string]struct{}{}
	if cfg.RootVer {
		for _, p := range pvs {
			for k := range p.root {
				if dfRoot[k] == nil {
					dfRoot[k] = map[string]struct{}{}
				}
				dfRoot[k][p.pkg.PackageName] = struct{}{}
			}
		}
	}
	// pick: IDF-weighted fraction of a version's low-df keys found in the app; ties -> more hits -> newer
	pick := func(vs []*pvKeys, keysOf func(*pvKeys) map[nkey]struct{}, df map[nkey]map[string]struct{}) (*pvKeys, float64, int) {
		idf := func(k nkey) float64 { return math.Log(1 + N/float64(len(df[k]))) }
		var best *pvKeys
		bestSc, bestM := -1.0, 0
		for _, p := range vs {
			var wAll, wHit float64
			m, tot := 0, 0
			maxdf := 1
			if cfg.LowUniq > 0 {
				u := 0
				for k := range keysOf(p) {
					if len(df[k]) == 1 {
						u++
					}
				}
				if u <= cfg.LowUniq {
					maxdf = cfg.LowDf
				}
			}
			for k := range keysOf(p) {
				if len(df[k]) > maxdf {
					continue
				}
				w := idf(k)
				wAll += w
				tot++
				if _, ok := A[k]; ok {
					wHit += w
					m++
				}
			}
			if tot == 0 {
				continue
			}
			sc := wHit / wAll
			if sc > bestSc || (sc == bestSc && m > bestM) || (sc == bestSc && m == bestM && best != nil && vless(best.pkg.PackageVersion, p.pkg.PackageVersion)) {
				best, bestSc, bestM = p, sc, m
			}
		}
		return best, bestSc, bestM
	}
	var out []*MatchResults
	for n, vs := range byPkg {
		best, bestSc, bestM := pick(vs, func(p *pvKeys) map[nkey]struct{} { return p.keys }, df)
		if best == nil || bestM == 0 {
			continue
		}
		t, ok := cfg.Thresholds[n]
		if !ok {
			t = cfg.Thresholds["_global"]
		}
		if bestSc < t {
			continue
		}
		if cfg.RootVer {
			if rb, _, _ := pick(vs, func(p *pvKeys) map[nkey]struct{} { return p.root }, dfRoot); rb != nil {
				best = rb
			}
		}
		if cfg.TieBreak {
			ap := map[nkey]struct{}{}
			for _, p := range vs {
				for k := range p.keys {
					if _, ok := A[k]; ok {
						ap[k] = struct{}{}
					}
				}
			}
			bf := -1.0
			for _, p := range vs {
				tp := 0
				for k := range p.keys {
					if _, ok := ap[k]; ok {
						tp++
					}
				}
				f := 0.0
				if tp > 0 {
					f = 2 * float64(tp) / float64(2*tp+(len(ap)-tp)+(len(p.keys)-tp))
				}
				if f > bf {
					bf, best = f, p
				}
			}
		}
		out = append(out, &MatchResults{PackageInfo: best.pkg, CombinedMatches: bestM, TotalPackageHashes: len(best.keys),
			TotalAppHashes: len(A), TotalMatchPercentage: bestSc * 100})
	}
	if cfg.Dep > 0 {
		out = addDeps(out, byPkg, A, cfg)
	}
	if cfg.Floor > 0 || len(cfg.Floors) > 0 {
		kept := out[:0]
		for _, r := range out {
			f, ok := cfg.Floors[r.PackageInfo.PackageName]
			if !ok {
				f = cfg.Floor
			}
			if r.TotalMatchPercentage/100 >= f {
				kept = append(kept, r)
			}
		}
		out = kept
	}
	if cfg.VsetMargin >= 0 {
		for _, r := range out {
			r.VersionSet = versionSet(r, byPkg, A, df, dfRoot, N, cfg)
		}
	}
	if f := os.Getenv("HEDIS_HITS_OUT"); f != "" {
		writeHits(f, pairs, out, byPkg, df, cfg)
	}
	if f := os.Getenv("HEDIS_VSET_OUT"); f != "" {
		writeVset(f, out, byPkg, A, df, dfRoot, N, cfg)
	}
	return out, nil
}

// writeHits (eval only, HEDIS_HITS_OUT): for each predicted package@version, the app function indices
// whose key is one of that version's keys, with the key's package df (for post-hoc DF / clustering filters).
func writeHits(f string, pairs [][2]string, out []*MatchResults, byPkg map[string][]*pvKeys, df map[nkey]map[string]struct{}, cfg NormMatchConfig) {
	type hit struct {
		I  int    `json:"i"`
		K  string `json:"k"`
		DF int    `json:"df"`
	}
	type rec struct {
		Pkg   string `json:"package"`
		Ver   string `json:"version"`
		NKeys int    `json:"nkeys"`
		NUniq int    `json:"nuniq"`
		Hits  []hit  `json:"hits"`
	}
	var recs []rec
	for _, r := range out {
		var ks map[nkey]struct{}
		for _, p := range byPkg[r.PackageInfo.PackageName] {
			if p.pkg.PackageVersion == r.PackageInfo.PackageVersion {
				ks = p.keys
			}
		}
		u := 0
		for k := range ks {
			if len(df[k]) == 1 {
				u++
			}
		}
		x := rec{Pkg: r.PackageInfo.PackageName, Ver: r.PackageInfo.PackageVersion, NKeys: len(ks), NUniq: u}
		for i, pr := range pairs {
			k := cfg.key(pr[0], pr[1])
			if _, ok := ks[k]; ok {
				x.Hits = append(x.Hits, hit{i, k.s + "|" + k.c, len(df[k])})
			}
		}
		recs = append(recs, x)
	}
	if b, err := json.Marshal(recs); err == nil {
		os.WriteFile(f, b, 0o644)
	}
}

type verScore struct {
	V       string   `json:"v"`
	Sc      float64  `json:"sc"`
	M       int      `json:"m"`
	Tot     int      `json:"tot"`
	NKeys   int      `json:"nkeys"`
	SameAll bool     `json:"same_all"`
	Hit     []string `json:"hit"`
}

// versionScores: every DB version of r's package scored as the version selector does (pick formula on root
// keys with dfRoot when RootVer, else all keys), with its sorted app-matched selector keys.
func versionScores(r *MatchResults, byPkg map[string][]*pvKeys, A map[nkey]struct{}, df, dfRoot map[nkey]map[string]struct{}, N float64, cfg NormMatchConfig) []verScore {
	vs := byPkg[r.PackageInfo.PackageName]
	keysOf := func(p *pvKeys) map[nkey]struct{} { return p.keys }
	d := df
	if cfg.RootVer {
		keysOf = func(p *pvKeys) map[nkey]struct{} { return p.root }
		d = dfRoot
	}
	var chosen *pvKeys
	for _, p := range vs {
		if p.pkg.PackageVersion == r.PackageInfo.PackageVersion {
			chosen = p
		}
	}
	var res []verScore
	for _, p := range vs {
		maxdf := 1
		if cfg.LowUniq > 0 {
			u := 0
			for k := range keysOf(p) {
				if len(d[k]) == 1 {
					u++
				}
			}
			if u <= cfg.LowUniq {
				maxdf = cfg.LowDf
			}
		}
		var wAll, wHit float64
		v := verScore{V: p.pkg.PackageVersion, NKeys: len(p.keys)}
		for k := range keysOf(p) {
			if len(d[k]) > maxdf {
				continue
			}
			w := math.Log(1 + N/float64(len(d[k])))
			wAll += w
			v.Tot++
			if _, ok := A[k]; ok {
				wHit += w
				v.M++
				v.Hit = append(v.Hit, k.s+"|"+k.c)
			}
		}
		if wAll > 0 {
			v.Sc = wHit / wAll
		}
		sort.Strings(v.Hit)
		if chosen != nil && len(chosen.keys) == len(p.keys) {
			v.SameAll = true
			for k := range p.keys {
				if _, ok := chosen.keys[k]; !ok {
					v.SameAll = false
					break
				}
			}
		}
		res = append(res, v)
	}
	return res
}

// versionSet: the chosen version plus near-tied (score >= chosen - VsetMargin) and, with VsetSameKeys,
// identical-matched-key-set versions; sorted by version.
func versionSet(r *MatchResults, byPkg map[string][]*pvKeys, A map[nkey]struct{}, df, dfRoot map[nkey]map[string]struct{}, N float64, cfg NormMatchConfig) []string {
	vs := versionScores(r, byPkg, A, df, dfRoot, N, cfg)
	var c *verScore
	for i := range vs {
		if vs[i].V == r.PackageInfo.PackageVersion {
			c = &vs[i]
		}
	}
	set := []string{r.PackageInfo.PackageVersion}
	if c == nil {
		return set
	}
	ck := strings.Join(c.Hit, ",")
	for _, v := range vs {
		if v.V == c.V {
			continue
		}
		if v.Sc >= c.Sc-cfg.VsetMargin-1e-12 || (cfg.VsetSameKeys && strings.Join(v.Hit, ",") == ck) {
			set = append(set, v.V)
		}
	}
	sort.Slice(set, func(i, j int) bool { return vless(set[i], set[j]) })
	return set
}

// writeVset (M35, HEDIS_VSET_OUT): for each predicted package, every DB version with the score the version
// selector computes (pick formula on root keys with dfRoot when RootVer, else all keys), its hit count, and the
// sorted list of app-matched scoring keys (so versions with identical matched-key sets can be grouped), plus
// whether the version's full key set equals the chosen version's. Read-only, eval only.
func writeVset(f string, out []*MatchResults, byPkg map[string][]*pvKeys, A map[nkey]struct{}, df, dfRoot map[nkey]map[string]struct{}, N float64, cfg NormMatchConfig) {
	type rec struct {
		Pkg  string     `json:"package"`
		Ver  string     `json:"version"`
		Sc   float64    `json:"score"`
		Vers []verScore `json:"versions"`
	}
	var recs []rec
	for _, r := range out {
		recs = append(recs, rec{r.PackageInfo.PackageName, r.PackageInfo.PackageVersion, r.TotalMatchPercentage / 100, versionScores(r, byPkg, A, df, dfRoot, N, cfg)})
	}
	if b, err := json.Marshal(recs); err == nil {
		os.WriteFile(f, b, 0o644)
	}
}

func addDeps(out []*MatchResults, byPkg map[string][]*pvKeys, A map[nkey]struct{}, cfg NormMatchConfig) []*MatchResults {
	have := map[string]bool{}
	keysOf := map[string]map[nkey]struct{}{}
	for _, r := range out {
		have[r.PackageInfo.PackageName] = true
	}
	for n, vs := range byPkg {
		for _, p := range vs {
			keysOf[n+"@"+p.pkg.PackageVersion] = p.keys
		}
	}
	for i := 0; i < len(out); i++ { // grows: implied packages can imply further ones
		q := out[i].PackageInfo
		KQ := keysOf[q.PackageName+"@"+q.PackageVersion]
		for _, dn := range cfg.Deps[q.PackageName][q.PackageVersion] {
			if have[dn] || byPkg[dn] == nil {
				continue
			}
			var best *pvKeys
			bestHit, bestOK := -1, false
			bestCov := 0.0
			for _, p := range byPkg[dn] {
				sh, shHit, hit := 0, 0, 0
				for k := range p.keys {
					_, inA := A[k]
					if inA {
						hit++
					}
					if _, inQ := KQ[k]; inQ {
						sh++
						if inA {
							shHit++
						}
					}
				}
				if sh < 5 {
					continue
				}
				cov := float64(shHit) / float64(sh)
				ok := cov >= cfg.Dep
				if best == nil || (ok && !bestOK) || (ok == bestOK && (hit > bestHit || (hit == bestHit && vless(best.pkg.PackageVersion, p.pkg.PackageVersion)))) {
					best, bestHit, bestOK, bestCov = p, hit, ok, cov
				}
			}
			if best != nil && bestOK {
				have[dn] = true
				out = append(out, &MatchResults{PackageInfo: best.pkg, CombinedMatches: bestHit, TotalPackageHashes: len(best.keys),
					TotalAppHashes: len(A), TotalMatchPercentage: bestCov * 100})
			}
		}
	}
	return out
}

func loadNormKeys(rn string, cfg NormMatchConfig, base map[nkey]struct{}) ([]*pvKeys, error) {
	ctx := context.Background()
	dbName := os.Getenv("HEDIS_PACKAGES_DB")
	if dbName == "" {
		dbName = "clean"
	}
	cl, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_CONNECTION_STRING")))
	if err != nil {
		return nil, err
	}
	defer cl.Disconnect(ctx)
	db := cl.Database(dbName)
	// only completed (package, rn) jobs
	done := map[string]bool{}
	dc, err := db.Collection("fp_done").Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return nil, err
	}
	var drows []struct {
		K string `bson:"_id"`
	}
	dc.All(ctx, &drows)
	for _, d := range drows {
		done[d.K] = true
	}
	// M35: HEDIS_MAX_OID (24-hex ObjectId) pins the DB to its state at that time: hash rows and packages
	// inserted later are ignored (read-only reproducibility while another job adds versions).
	var maxOID primitive.ObjectID
	if s := os.Getenv("HEDIS_MAX_OID"); s != "" {
		if o, err := primitive.ObjectIDFromHex(s); err == nil {
			maxOID = o
		}
	}
	// M37: RN fallback when the app's RN has no fingerprints (nearest or union, see rnfallback.go)
	rns := resolveRNs(ctx, db, rn, maxOID)
	if len(rns) == 0 {
		return nil, nil
	}
	hq := bson.M{"react_native_version": bson.M{"$in": rns}}
	if !maxOID.IsZero() {
		hq["_id"] = bson.M{"$lte": maxOID}
	}
	cur, err := db.Collection("hashes").Find(ctx, hq,
		options.Find().SetProjection(bson.M{"_id": 0, "package_id": 1, "react_native_version": 1, "ns": 1, "nc": 1, "sub": 1}).SetBatchSize(50000))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	groups := map[primitive.ObjectID]*pvKeys{}
	var ids []primitive.ObjectID
	for cur.Next(ctx) {
		var d struct {
			P   primitive.ObjectID `bson:"package_id"`
			NS  string             `bson:"ns"`
			NC  string             `bson:"nc"`
			Sub bool               `bson:"sub"`
			RN  string             `bson:"react_native_version"`
		}
		if err := cur.Decode(&d); err != nil {
			return nil, err
		}
		if !done[d.P.Hex()+"|"+d.RN] {
			continue
		}
		g := groups[d.P]
		if g == nil {
			g = &pvKeys{keys: map[nkey]struct{}{}, root: map[nkey]struct{}{}}
			groups[d.P] = g
			ids = append(ids, d.P)
		}
		k := cfg.key(d.NS, d.NC)
		if _, in := base[k]; !in {
			g.keys[k] = struct{}{}
			if !d.Sub {
				g.root[k] = struct{}{}
			}
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	pq := bson.M{"_id": bson.M{"$in": ids}}
	if !maxOID.IsZero() {
		pq = bson.M{"$and": []bson.M{{"_id": bson.M{"$in": ids}}, {"_id": bson.M{"$lte": maxOID}}}}
	}
	pc, err := db.Collection("packages").Find(ctx, pq)
	if err != nil {
		return nil, err
	}
	var ps []models.PackageWithVulnerabilitiesModel
	if err := pc.All(ctx, &ps); err != nil {
		return nil, err
	}
	var out []*pvKeys
	for i := range ps {
		if g := groups[ps[i].ID]; g != nil {
			g.pkg = &ps[i]
			out = append(out, g)
		}
	}
	return out, nil
}
