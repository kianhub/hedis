package pipeline

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"hermes-decompiler/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// M37: RN-version fallback. When the app's React Native version has no fingerprint rows in the DB
// (e.g. app on RN 0.72 while the DB only holds 0.74/0.76/0.79), the matchers used to send `$in: null`
// to mongo and crash with "(BadValue) $in needs an array". Now they fall back to other RN versions.
//
// Mode (flag --rn-fallback or env HEDIS_RN_FALLBACK): "nearest" (default) = the covered RN version
// with the smallest numeric distance (ties -> the newer one); "union" = all covered RN versions.
// HEDIS_EXCLUDE_RN (comma list, eval only) hides RN versions at read time (leave-own-RN-out test).
var RNFallback = ""

func rnFallbackMode() string {
	m := RNFallback
	if m == "" {
		m = os.Getenv("HEDIS_RN_FALLBACK")
	}
	if m != "union" {
		m = "nearest"
	}
	return m
}

func excludedRNs() map[string]bool {
	ex := map[string]bool{}
	for _, s := range strings.Split(os.Getenv("HEDIS_EXCLUDE_RN"), ",") {
		if s = strings.TrimSpace(s); s != "" {
			ex[s] = true
		}
	}
	return ex
}

// rnNum maps "0.72" / "0.72.4" / "rn072" to a comparable float (major*1000+minor+patch/1000).
func rnNum(v string) (float64, bool) {
	v = strings.TrimPrefix(v, "rn")
	if !strings.Contains(v, ".") && len(v) == 3 { // rn072
		v = v[:1] + "." + v[1:]
	}
	p := strings.Split(v, ".")
	if len(p) < 2 {
		return 0, false
	}
	ma, e1 := strconv.Atoi(p[0])
	mi, e2 := strconv.Atoi(p[1])
	if e1 != nil || e2 != nil {
		return 0, false
	}
	f := float64(ma*1000 + mi)
	if len(p) > 2 {
		if pa, err := strconv.Atoi(p[2]); err == nil {
			f += float64(pa) / 1000
		}
	}
	return f, true
}

// ChooseRNVersions returns the RN versions whose fingerprints should be used for an app on rn,
// given the covered versions avail, plus a note describing any fallback ("" if none).
func ChooseRNVersions(rn string, avail []string, mode string) ([]string, string) {
	for _, a := range avail {
		if a == rn {
			return []string{rn}, ""
		}
	}
	if len(avail) == 0 {
		return nil, fmt.Sprintf("RN %s: no fingerprints for any RN version", rn)
	}
	s := append([]string(nil), avail...)
	sort.Slice(s, func(i, j int) bool { a, _ := rnNum(s[i]); b, _ := rnNum(s[j]); return a < b })
	if mode == "union" {
		return s, fmt.Sprintf("RN %s not in DB, fallback=union %v", rn, s)
	}
	x, ok := rnNum(rn)
	best, bd := s[len(s)-1], -1.0
	if ok {
		for _, a := range s {
			y, _ := rnNum(a)
			d := x - y
			if d < 0 {
				d = -d
			}
			if bd < 0 || d <= bd { // ascending order + <= : ties go to the newer version
				best, bd = a, d
			}
		}
	}
	return []string{best}, fmt.Sprintf("RN %s not in DB, fallback=nearest %s (covered %v)", rn, best, s)
}

// coveredRNs lists RN versions with at least one hash row (respecting the HEDIS_MAX_OID pin),
// minus HEDIS_EXCLUDE_RN. Candidates come from fp_done ids ("<pkgid>|<rn>") and baselines.
func coveredRNs(ctx context.Context, db *mongo.Database, maxOID primitive.ObjectID) ([]string, error) {
	cand := map[string]bool{}
	dc, err := db.Collection("fp_done").Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return nil, err
	}
	var drows []struct {
		K string `bson:"_id"`
	}
	dc.All(ctx, &drows)
	for _, d := range drows {
		if i := strings.LastIndex(d.K, "|"); i >= 0 {
			cand[d.K[i+1:]] = true
		}
	}
	for k := range rnBaselines {
		cand[k] = true
	}
	ex := excludedRNs()
	var out []string
	for rn := range cand {
		if ex[rn] {
			continue
		}
		q := bson.M{"react_native_version": rn}
		if !maxOID.IsZero() {
			q["_id"] = bson.M{"$lte": maxOID}
		}
		if n, err := db.Collection("hashes").CountDocuments(ctx, q, options.Count().SetLimit(1)); err == nil && n > 0 {
			out = append(out, rn)
		}
	}
	sort.Strings(out)
	return out, nil
}

// resolveRNs picks the RN versions to load for rn and logs any fallback (stderr log + HEDIS_RN_LOG file).
func resolveRNs(ctx context.Context, db *mongo.Database, rn string, maxOID primitive.ObjectID) []string {
	avail, err := coveredRNs(ctx, db, maxOID)
	if err != nil {
		utils.F.Printf("RN fallback: listing covered RN versions failed: %v\n", err)
		return []string{rn}
	}
	use, note := ChooseRNVersions(rn, avail, rnFallbackMode())
	if note != "" {
		utils.L.Printf("%s\n", note)
		if f := os.Getenv("HEDIS_RN_LOG"); f != "" {
			if fh, err := os.OpenFile(f, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
				fmt.Fprintln(fh, note)
				fh.Close()
			}
		}
		LastRNFallback = note
	}
	return use
}

// LastRNFallback holds the last fallback note (written to the JSON output as rn_fallback).
var LastRNFallback = ""
