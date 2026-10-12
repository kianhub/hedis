package pipeline

import (
	"context"
	"math"
	"os"

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
	A := map[nkey]struct{}{}
	for _, h := range app {
		ns, nc := normhash.Hashes(h.Structural.Raw, h.ContentAR1.Raw, h.ContentAR2.Raw)
		k := cfg.key(ns, nc)
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
	idf := func(k nkey) float64 { return math.Log(1 + N/float64(len(df[k]))) }
	var out []*MatchResults
	for n, vs := range byPkg {
		var best *pvKeys
		bestSc, bestM := -1.0, 0
		for _, p := range vs {
			var wAll, wHit float64
			m, tot := 0, 0
			for k := range p.keys {
				if len(df[k]) != 1 {
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
			if sc > bestSc || (sc == bestSc && m > bestM) {
				best, bestSc, bestM = p, sc, m
			}
		}
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
	return out, nil
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
		options.Find().SetProjection(bson.M{"_id": 0, "package_id": 1, "ns": 1, "nc": 1}).SetHint("cover_rn_pkg_norm").SetBatchSize(50000))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	groups := map[primitive.ObjectID]*pvKeys{}
	var ids []primitive.ObjectID
	for cur.Next(ctx) {
		var d struct {
			P  primitive.ObjectID `bson:"package_id"`
			NS string             `bson:"ns"`
			NC string             `bson:"nc"`
		}
		if err := cur.Decode(&d); err != nil {
			return nil, err
		}
		if !done[d.P.Hex()+"|"+rn] {
			continue
		}
		g := groups[d.P]
		if g == nil {
			g = &pvKeys{keys: map[nkey]struct{}{}}
			groups[d.P] = g
			ids = append(ids, d.P)
		}
		k := cfg.key(d.NS, d.NC)
		if _, in := base[k]; !in {
			g.keys[k] = struct{}{}
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
