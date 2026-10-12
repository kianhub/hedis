package pipeline

import (
	"context"
	"math"
	"os"
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
	return out, nil
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
	cur, err := db.Collection("hashes").Find(ctx, bson.M{"react_native_version": rn},
		options.Find().SetProjection(bson.M{"_id": 0, "package_id": 1, "ns": 1, "nc": 1, "sub": 1}).SetBatchSize(50000))
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
		}
		if err := cur.Decode(&d); err != nil {
			return nil, err
		}
		if !done[d.P.Hex()+"|"+rn] {
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
	pc, err := db.Collection("packages").Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
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
