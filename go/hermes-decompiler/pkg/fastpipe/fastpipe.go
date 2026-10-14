// Package fastpipe is a parallel, cached re-implementation of the package fingerprinting
// pipeline (npm install -> Metro -> hermesc -> disassemble -> 3-AR hashes -> baseline filter -> DB).
// It writes the same schema the thesis matcher (`packages -d`) reads:
//
//	packages:  models.PackageWithVulnerabilitiesModel
//	hashes:    models.FunctionObjectHashModel (one doc per function, baseline-filtered)
//	baselines (+ baselines_v3): models.BaselineModel
//
// Optimizations vs pipeline/clean.go:
//   - N worker copies of each RN env, jobs run in parallel
//   - npm install --no-save --prefer-offline --ignore-scripts (no separate uninstall; npm prunes)
//   - Metro transform cache kept warm (resetCache=false) per worker
//   - bundle-content dedupe: identical Metro output -> reuse hbc/hashes (sha256 keyed disk cache)
//   - baseline filter via hash set (O(n)) instead of nested loop (O(n*m))
//   - resume: skip (package, rn) pairs already present in DB
package fastpipe

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hermes-decompiler/pkg/database/models"
	"hermes-decompiler/pkg/hbc"
	"hermes-decompiler/pkg/normhash"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Config struct {
	MongoURI, DBName string
	EnvsRoot         string   // contains rnXXX template dirs
	RNs              []string // e.g. rn074, rn076
	Workers          int      // per RN version
	CacheDir         string
	StatsPath        string
	OSHermes         string
	Sub              bool // bundle the package subpath entry points (exports map / top-level files) in one entry; rows tagged sub, root rows untouched
	Release          bool // dev=false, minify=true bundles (store-app config); baseline filter on normalized key; raw IR not stored
}

type Job struct{ Name, Version, RN string }

type Stat struct {
	Job
	OK                                      bool   `json:"ok"`
	Err                                     string `json:"err,omitempty"`
	Skipped                                 string `json:"skipped,omitempty"`
	Forced                                  bool   `json:"forced,omitempty"`
	Install, Metro, Hermes, Hash, Db, Total float64
	Funcs, Kept                             int
	BundleSHA                               string
	CacheHit                                bool
	Subpaths                                []string `json:"subpaths,omitempty"`
}

func rnNum(rn string) string { // rn074 -> 0.74
	var n int
	fmt.Sscanf(strings.TrimPrefix(rn, "rn"), "%d", &n)
	return fmt.Sprintf("%.2f", float64(n)/100)
}

type key3 struct{ s, c1, c2 string }

type Runner struct {
	subFail  map[string]int // "<pkg>/<subpath>" -> versions where it failed to bundle alone (guarded by mu)
	cfg      Config
	db       *mongo.Database
	baseline map[string]map[key3]struct{} // rnNum -> set
	pkgIDs   map[string]primitive.ObjectID
	done     map[string]bool // pkgid|rn
	mu       sync.Mutex
	statsF   *os.File
}

func New(cfg Config) (*Runner, error) {
	cl, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		return nil, err
	}
	r := &Runner{cfg: cfg, db: cl.Database(cfg.DBName), baseline: map[string]map[key3]struct{}{},
		pkgIDs: map[string]primitive.ObjectID{}, done: map[string]bool{}}
	if cfg.StatsPath != "" {
		r.statsF, _ = os.OpenFile(cfg.StatsPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	}
	ctx := context.Background()
	r.db.Collection("hashes").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "react_native_version", Value: 1}, {Key: "package_id", Value: 1}}},
		{Keys: bson.D{{Key: "structural.hash", Value: 1}}},
	})
	r.db.Collection("packages").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "package_name", Value: 1}, {Key: "package_version", Value: 1}}, Options: options.Index().SetUnique(true)})
	// covering index for the matcher's $match{rn}+$group{package_id,3 hashes}: avoids fetching docs with raw IR
	r.db.Collection("hashes").Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "react_native_version", Value: 1},
		{Key: "package_id", Value: 1}, {Key: "structural.hash", Value: 1}, {Key: "ar1.hash", Value: 1}, {Key: "ar2.hash", Value: 1}},
		Options: options.Index().SetName("cover_rn_pkg_3hash")})
	r.db.Collection("hashes").Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "react_native_version", Value: 1},
		{Key: "package_id", Value: 1}, {Key: "ns", Value: 1}, {Key: "nc", Value: 1}}, Options: options.Index().SetName("cover_rn_pkg_norm")})
	releaseMode = cfg.Release
	// load existing packages + done pairs for resume
	cur, err := r.db.Collection("packages").Find(ctx, bson.M{})
	if err == nil {
		var ps []models.PackageWithVulnerabilitiesModel
		cur.All(ctx, &ps)
		for _, p := range ps {
			r.pkgIDs[p.PackageName+"@"+p.PackageVersion] = p.ID
		}
	}
	// fp_done is the only completion marker (written after the hash insert succeeds)
	cur2, err := r.db.Collection("fp_done").Find(ctx, bson.M{})
	if err == nil {
		var rows []struct {
			K string `bson:"_id"`
		}
		cur2.All(ctx, &rows)
		for _, x := range rows {
			r.done[x.K] = true
		}
	}
	return r, nil
}

func (r *Runner) logStat(s *Stat) {
	if r.statsF == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	b, _ := json.Marshal(s)
	r.statsF.Write(append(b, '\n'))
}

func run(dir string, timeout time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	c := exec.CommandContext(ctx, name, args...)
	c.Dir = dir
	out, err := c.CombinedOutput()
	if err != nil {
		o := string(out)
		if len(o) > 600 {
			o = o[len(o)-600:]
		}
		return o, fmt.Errorf("%s %v: %v: %s", name, args, err, o)
	}
	return string(out), nil
}

func (r *Runner) hermesc(env string) string {
	return filepath.Join(env, "node_modules", "react-native", "sdks", "hermesc", r.cfg.OSHermes, "hermesc")
}

const bundleScript = `import Metro from 'metro';
const config = await Metro.loadConfig();
config.resetCache = %v;
config.maxWorkers = 2;
// M34: pinned peers installed only to satisfy npm are resolved as empty modules, so their code stays out of this package's keys
const empty = new Set(%s);
if (empty.size) {
  const prev = config.resolver.resolveRequest;
  config.resolver.resolveRequest = (ctx, name, platform) => {
    const root = name.startsWith('@') ? name.split('/').slice(0, 2).join('/') : name.split('/')[0];
    if (empty.has(root)) return { type: 'empty' };
    return prev ? prev(ctx, name, platform) : ctx.resolveRequest(ctx, name, platform);
  };
}
await Metro.runBuild(config, { entry: '%s', platform: 'ios', dev: %v, minify: %v, out: '%s' });`

var releaseMode bool

func writeBundleScript(env, entry, out string, reset bool, empty ...string) error {
	if empty == nil {
		empty = []string{}
	}
	ej, _ := json.Marshal(empty)
	return os.WriteFile(filepath.Join(env, "bundle-fast.mjs"), []byte(fmt.Sprintf(bundleScript, reset, string(ej), entry, !releaseMode, releaseMode, out)), 0644)
}

func hashesFromHBC(path string) ([]*models.FunctionObjectHashModel, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	rd := hbc.NewHBCReader()
	if err := rd.ReadWholeFile(f); err != nil {
		return nil, "", err
	}
	fos, err := hbc.CreateFunctionObjects(rd)
	if err != nil {
		return nil, "", err
	}
	out := make([]*models.FunctionObjectHashModel, 0, len(fos))
	for _, fo := range fos {
		s, c1, c2 := fo.ToIR()
		h := &models.FunctionObjectHashModel{Structural: models.FunctionObjectDataModel{Raw: s, Hash: sha(s)}}
		if c1 != "" {
			h.ContentAR1 = models.FunctionObjectDataModel{Raw: c1, Hash: sha(c1)}
		}
		if c2 != "" {
			h.ContentAR2 = models.FunctionObjectDataModel{Raw: c2, Hash: sha(c2)}
		}
		h.NormS, h.NormC = normhash.Hashes(s, c1, c2)
		out = append(out, h)
	}
	return out, fmt.Sprintf("%d", rd.Header.Version), nil
}

func sha(s string) string { x := sha256.Sum256([]byte(s)); return hex.EncodeToString(x[:]) }

// EnsureBaseline builds (or loads) the baseline for an RN env and stores it in both
// "baselines" (read by the matcher) and "baselines_v3" (written by the original pipeline).
func (r *Runner) EnsureBaseline(rn string) error {
	ctx := context.Background()
	v := rnNum(rn)
	var b models.BaselineModel
	err := r.db.Collection("baselines").FindOne(ctx, bson.M{"react_native_version": v}).Decode(&b)
	if err == mongo.ErrNoDocuments {
		env := filepath.Join(r.cfg.EnvsRoot, rn)
		out := "baseline_out_fast.js"
		if err := writeBundleScript(env, "baseline_entry.js", out, true); err != nil {
			return err
		}
		if _, err := run(env, 10*time.Minute, "node", "bundle-fast.mjs"); err != nil {
			return err
		}
		if _, err := run(env, 10*time.Minute, r.hermesc(env), "-O", "-emit-binary", "-out", "baseline-fast.hbc", out); err != nil {
			return err
		}
		hs, ver, err := hashesFromHBC(filepath.Join(env, "baseline-fast.hbc"))
		if err != nil {
			return err
		}
		b = models.BaselineModel{ID: primitive.NewObjectID(), ReactNativeVersion: v, HermesBytecodeVersion: ver}
		for i, h := range hs {
			b.Hashes = append(b.Hashes, models.Hash{RelativeFunctionIndex: i, StructuralHash: h.Structural.Hash,
				ContentIR1Hash: h.ContentAR1.Hash, ContentIR2Hash: h.ContentAR2.Hash, NormS: h.NormS, NormC: h.NormC})
		}
		if _, err := r.db.Collection("baselines").InsertOne(ctx, b); err != nil {
			return err
		}
		r.db.Collection("baselines_v3").InsertOne(ctx, b)
	} else if err != nil {
		return err
	}
	set := make(map[key3]struct{}, len(b.Hashes))
	for _, h := range b.Hashes {
		if r.cfg.Release {
			set[key3{h.NormS, h.NormC, "n"}] = struct{}{}
		} else {
			set[key3{h.StructuralHash, h.ContentIR1Hash, h.ContentIR2Hash}] = struct{}{}
		}
	}
	r.mu.Lock()
	r.baseline[v] = set
	r.mu.Unlock()
	return nil
}

func (r *Runner) pkgID(name, ver string) (primitive.ObjectID, error) {
	k := name + "@" + ver
	r.mu.Lock()
	id, ok := r.pkgIDs[k]
	r.mu.Unlock()
	if ok {
		return id, nil
	}
	p := models.PackageWithVulnerabilitiesModel{ID: primitive.NewObjectID(), PackageName: name, PackageVersion: ver, AdvisoryUrls: []string{}}
	_, err := r.db.Collection("packages").InsertOne(context.Background(), p)
	if err != nil {
		var ex models.PackageWithVulnerabilitiesModel
		if e2 := r.db.Collection("packages").FindOne(context.Background(), bson.M{"package_name": name, "package_version": ver}).Decode(&ex); e2 != nil {
			return id, err
		}
		p.ID = ex.ID
	}
	r.mu.Lock()
	r.pkgIDs[k] = p.ID
	r.mu.Unlock()
	return p.ID, nil
}

// prepareWorkers clones the RN env template once per worker (cp -a, reused across runs).
func (r *Runner) prepareWorkers(rn string) ([]string, error) {
	var dirs []string
	for w := 0; w < r.cfg.Workers; w++ {
		d := filepath.Join(r.cfg.EnvsRoot, "work", fmt.Sprintf("%s_w%d", rn, w))
		if _, err := os.Stat(filepath.Join(d, "node_modules")); err != nil {
			os.MkdirAll(filepath.Dir(d), 0755)
			os.RemoveAll(d)
			if _, err := run("/", 20*time.Minute, "cp", "-a", filepath.Join(r.cfg.EnvsRoot, rn), d); err != nil {
				return nil, err
			}
		}
		dirs = append(dirs, d)
	}
	return dirs, nil
}

func (r *Runner) Run(jobs []Job) error {
	byRN := map[string][]Job{}
	for _, j := range jobs {
		byRN[j.RN] = append(byRN[j.RN], j)
	}
	var wg sync.WaitGroup
	for rn, js := range byRN {
		if err := r.EnsureBaseline(rn); err != nil {
			return fmt.Errorf("baseline %s: %w", rn, err)
		}
		dirs, err := r.prepareWorkers(rn)
		if err != nil {
			return err
		}
		// group by package name so a worker keeps one package's deps warm across versions
		groups := map[string][]Job{}
		var order []string
		for _, j := range js {
			if _, ok := groups[j.Name]; !ok {
				order = append(order, j.Name)
			}
			groups[j.Name] = append(groups[j.Name], j)
		}
		q := make(chan []Job, len(order))
		for _, n := range order {
			q <- groups[n]
		}
		close(q)
		for _, d := range dirs {
			wg.Add(1)
			go func(env string) {
				defer wg.Done()
				for g := range q {
					for _, j := range g {
						r.one(env, j)
					}
				}
			}(d)
		}
	}
	wg.Wait()
	return nil
}

func (r *Runner) one(env string, j Job) {
	st := &Stat{Job: j}
	t0 := time.Now()
	defer func() { st.Total = time.Since(t0).Seconds(); r.logStat(st) }()
	v := rnNum(j.RN)
	id, err := r.pkgID(j.Name, j.Version)
	if err != nil {
		st.Err = err.Error()
		return
	}
	dk := id.Hex() + "|" + v
	if r.cfg.Sub {
		dk += "|sub"
	}
	r.mu.Lock()
	isDone := r.done[dk]
	r.mu.Unlock()
	if isDone {
		st.OK, st.Skipped = true, "already-in-db"
		return
	}
	t := time.Now()
	npmArgs := []string{"install", j.Name + "@" + j.Version, "--no-save", "--no-audit", "--no-fund", "--ignore-scripts", "--prefer-offline", "--loglevel=error"}
	pins := []string{}
	if b, err := os.ReadFile(filepath.Join(env, "peer-pins.txt")); err == nil {
		for _, pin := range strings.Fields(string(b)) {
			if i := strings.LastIndex(pin, "@"); i > 0 && pin[:i] != j.Name {
				pins = append(pins, pin)
			}
		}
	}
	// M34: an env installed with legacy-peer-deps (rn079) never gets peers, so its pins go on every install
	firstArgs := npmArgs
	var empty []string // peers installed from pins: bundled as empty modules
	pinNames := func() []string {
		n := []string{}
		for _, p := range pins {
			n = append(n, p[:strings.LastIndex(p, "@")])
		}
		return n
	}
	if b, err := os.ReadFile(filepath.Join(env, ".npmrc")); err == nil && strings.Contains(string(b), "legacy-peer-deps=true") {
		firstArgs = append(append([]string{}, npmArgs...), pins...)
		empty = pinNames()
	}
	if _, err := run(env, 10*time.Minute, "npm", firstArgs...); err != nil {
		// M34: a peer-dependency conflict (ERESOLVE) is retried with --force, which keeps the env's peer tree
		// (metro etc.); legacy-peer-deps would prune it in the rn074/rn076 envs. The version guard below still applies.
		if !strings.Contains(err.Error(), "ERESOLVE") && !strings.Contains(err.Error(), "upstream dependency conflict") {
			st.Err = "install: " + err.Error()
			return
		}
		// peer-pins.txt in the env (one "name@version" per line) pins RN-compatible peers for the forced install, so
		// --force does not pull a latest peer that this RN version cannot transform (e.g. gesture-handler codegen specs).
		forceArgs := append(append([]string{}, npmArgs...), pins...)
		if _, err2 := run(env, 10*time.Minute, "npm", append(forceArgs, "--force")...); err2 != nil {
			st.Err = "install: " + err2.Error()
			return
		}
		st.Forced = true
		empty = pinNames()
	}
	st.Install = time.Since(t).Seconds()
	// guard: the version actually resolved in node_modules must be the requested one
	var pj struct {
		Version string `json:"version"`
	}
	if b, err := os.ReadFile(filepath.Join(env, "node_modules", j.Name, "package.json")); err != nil || json.Unmarshal(b, &pj) != nil || pj.Version != j.Version {
		st.Err = fmt.Sprintf("install: resolved version %q != %q", pj.Version, j.Version)
		return
	}
	entry := fmt.Sprintf("import * as ModuleToBundle from '%s';\n\nglobal.MODULE_TO_BUNDLE = ModuleToBundle;", j.Name)
	if r.cfg.Sub {
		subs := Subpaths(filepath.Join(env, "node_modules", j.Name))
		// skip subpaths that already failed to bundle alone in 2+ other versions of this package
		r.mu.Lock()
		kept := subs[:0]
		for _, s := range subs {
			if r.subFail[j.Name+"/"+s] < 2 {
				kept = append(kept, s)
			}
		}
		r.mu.Unlock()
		subs = kept
		st.Subpaths = subs
		if len(subs) == 0 {
			st.OK, st.Skipped = true, "no-subpaths"
			r.db.Collection("fp_done").UpdateOne(context.Background(), bson.M{"_id": dk}, bson.M{"$set": bson.M{"funcs": 0, "kept": 0}}, options.Update().SetUpsert(true))
			return
		}
		var b strings.Builder
		for i, s := range subs {
			fmt.Fprintf(&b, "import * as S%d from '%s/%s';\nglobal.S%d = S%d;\n", i, j.Name, s, i, i)
		}
		entry = b.String()
	}
	os.WriteFile(filepath.Join(env, "package_entry_fast.js"), []byte(entry), 0644)
	out := "package_out_fast.js"
	writeBundleScript(env, "package_entry_fast.js", out, false, empty...)
	t = time.Now()
	if _, err := run(env, 10*time.Minute, "node", "bundle-fast.mjs"); err != nil {
		// per-subpath probing costs one Metro run each: only for packages with few subpaths (crypto-js, zod)
		if !r.cfg.Sub || len(st.Subpaths) < 1 {
			st.Err = "metro: " + err.Error()
			return
		}
		// one bad subpath (node-only, type-only, unresolvable) fails the whole entry: keep the ones that bundle alone
		var ok []string
		for _, s := range st.Subpaths {
			e := fmt.Sprintf("import * as S0 from '%s/%s';\nglobal.S0 = S0;\n", j.Name, s)
			os.WriteFile(filepath.Join(env, "package_entry_fast.js"), []byte(e), 0644)
			if _, err := run(env, 5*time.Minute, "node", "bundle-fast.mjs"); err == nil {
				ok = append(ok, s)
			} else {
				r.mu.Lock()
				if r.subFail == nil {
					r.subFail = map[string]int{}
				}
				r.subFail[j.Name+"/"+s]++
				r.mu.Unlock()
			}
		}
		st.Subpaths = ok
		if len(ok) == 0 { // nothing bundles on this RN: record the same marker as a package without subpaths
			st.OK, st.Skipped = true, "no-subpaths"
			r.db.Collection("fp_done").UpdateOne(context.Background(), bson.M{"_id": dk}, bson.M{"$set": bson.M{"funcs": 0, "kept": 0, "no_subpath": true}}, options.Update().SetUpsert(true))
			r.mu.Lock()
			r.done[dk] = true
			r.mu.Unlock()
			return
		}
		var b strings.Builder
		for i, s := range ok {
			fmt.Fprintf(&b, "import * as S%d from '%s/%s';\nglobal.S%d = S%d;\n", i, j.Name, s, i, i)
		}
		os.WriteFile(filepath.Join(env, "package_entry_fast.js"), []byte(b.String()), 0644)
		if _, err := run(env, 10*time.Minute, "node", "bundle-fast.mjs"); err != nil {
			st.Err = "metro: " + err.Error()
			return
		}
	}
	st.Metro = time.Since(t).Seconds()
	bundle, err := os.ReadFile(filepath.Join(env, out))
	if err != nil {
		st.Err = err.Error()
		return
	}
	bsha := sha(string(bundle))
	st.BundleSHA = bsha
	cacheHBC := filepath.Join(r.cfg.CacheDir, j.RN, bsha+".hbc")
	t = time.Now()
	if _, err := os.Stat(cacheHBC); err == nil {
		st.CacheHit = true
	} else {
		os.MkdirAll(filepath.Dir(cacheHBC), 0755)
		tmp := filepath.Join(env, "package-fast.hbc")
		if _, err := run(env, 10*time.Minute, r.hermesc(env), "-O", "-emit-binary", "-out", tmp, out); err != nil {
			st.Err = "hermesc: " + err.Error()
			return
		}
		os.Rename(tmp, cacheHBC)
	}
	st.Hermes = time.Since(t).Seconds()
	t = time.Now()
	// no in-memory bundle cache: it held every bundle's raw IR and OOM-killed the build (6 GB RSS);
	// the on-disk .hbc cache already makes re-hashing cheap.
	hs, _, err := hashesFromHBC(cacheHBC)
	if err != nil {
		st.Err = "disasm: " + err.Error()
		return
	}
	st.Funcs = len(hs)
	base := r.baseline[v]
	docs := make([]interface{}, 0, len(hs))
	for _, h := range hs {
		bk := key3{h.Structural.Hash, h.ContentAR1.Hash, h.ContentAR2.Hash}
		if r.cfg.Release {
			bk = key3{h.NormS, h.NormC, "n"}
		}
		if _, inBase := base[bk]; inBase {
			continue
		}
		d := *h
		d.ID, d.PackageID, d.ReactNativeVersion = primitive.NewObjectID(), id, v
		d.Sub = r.cfg.Sub
		if r.cfg.Release {
			d.Structural.Raw, d.ContentAR1.Raw, d.ContentAR2.Raw = "", "", ""
		}
		docs = append(docs, d)
	}
	st.Kept = len(docs)
	st.Hash = time.Since(t).Seconds()
	t = time.Now()
	// idempotent: drop partial inserts from an interrupted previous run
	del := bson.M{"package_id": id, "react_native_version": v}
	if r.cfg.Sub {
		del["sub"] = true
	}
	r.db.Collection("hashes").DeleteMany(context.Background(), del)
	if len(docs) > 0 {
		if _, err := r.db.Collection("hashes").InsertMany(context.Background(), docs, options.InsertMany().SetOrdered(false)); err != nil {
			st.Err = "db: " + err.Error()
			return
		}
	}
	r.db.Collection("fp_done").UpdateOne(context.Background(), bson.M{"_id": dk},
		bson.M{"$set": bson.M{"funcs": len(hs), "kept": len(docs), "bundle_sha": bsha}}, options.Update().SetUpsert(true))
	r.mu.Lock()
	r.done[dk] = true
	r.mu.Unlock()
	st.Db = time.Since(t).Seconds()
	st.OK = true
}

// ImportFromStats rebuilds DB rows from the on-disk .hbc cache using a previous run's stats JSONL
// (ok rows carry BundleSHA), without npm/Metro. Used to recover after Mongo data loss.
func (r *Runner) ImportFromStats(statsPath string, rns []string) (int, error) {
	for _, rn := range rns {
		if err := r.EnsureBaseline(rn); err != nil {
			return 0, err
		}
	}
	f, err := os.ReadFile(statsPath)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, line := range strings.Split(string(f), "\n") {
		var s Stat
		if json.Unmarshal([]byte(line), &s) != nil || !s.OK || s.BundleSHA == "" {
			continue
		}
		v := rnNum(s.RN)
		id, err := r.pkgID(s.Name, s.Version)
		if err != nil {
			return n, err
		}
		dk := id.Hex() + "|" + v
		if r.done[dk] {
			continue
		}
		hs, _, err := hashesFromHBC(filepath.Join(r.cfg.CacheDir, s.RN, s.BundleSHA+".hbc"))
		if err != nil {
			continue
		}
		base := r.baseline[v]
		docs := make([]interface{}, 0, len(hs))
		for _, h := range hs {
			bk := key3{h.Structural.Hash, h.ContentAR1.Hash, h.ContentAR2.Hash}
			if r.cfg.Release {
				bk = key3{h.NormS, h.NormC, "n"}
			}
			if _, in := base[bk]; in {
				continue
			}
			d := *h
			d.ID, d.PackageID, d.ReactNativeVersion = primitive.NewObjectID(), id, v
			if r.cfg.Release {
				d.Structural.Raw, d.ContentAR1.Raw, d.ContentAR2.Raw = "", "", ""
			}
			docs = append(docs, d)
		}
		ctx := context.Background()
		r.db.Collection("hashes").DeleteMany(ctx, bson.M{"package_id": id, "react_native_version": v})
		if len(docs) > 0 {
			if _, err := r.db.Collection("hashes").InsertMany(ctx, docs, options.InsertMany().SetOrdered(false)); err != nil {
				return n, err
			}
		}
		r.db.Collection("fp_done").UpdateOne(ctx, bson.M{"_id": dk}, bson.M{"$set": bson.M{"funcs": len(hs), "kept": len(docs), "bundle_sha": s.BundleSHA}}, options.Update().SetUpsert(true))
		r.done[dk] = true
		n++
	}
	return n, nil
}

// Subpaths lists importable subpath entry points of an installed package: string-valued exports
// keys ("./v4", not "." / wildcards / json), else top-level .js files (UMD-style packages like
// crypto-js whose consumers import "crypto-js/aes"). Capped at 80.
func Subpaths(dir string) []string {
	b, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil
	}
	var pj struct {
		Exports json.RawMessage `json:"exports"`
		Main    string          `json:"main"`
	}
	json.Unmarshal(b, &pj)
	var out []string
	var ex map[string]json.RawMessage
	if len(pj.Exports) > 0 && json.Unmarshal(pj.Exports, &ex) == nil {
		for k := range ex {
			if !strings.HasPrefix(k, "./") || strings.Contains(k, "*") || strings.HasSuffix(k, ".json") || strings.HasSuffix(k, "/") || toolingFile(path.Base(k)) {
				continue
			}
			out = append(out, strings.TrimPrefix(k, "./"))
		}
	} else if len(pj.Exports) == 0 {
		main := strings.TrimPrefix(filepath.Clean(pj.Main), "./")
		ents, _ := os.ReadDir(dir)
		for _, e := range ents {
			n := e.Name()
			if e.IsDir() || !strings.HasSuffix(n, ".js") || strings.HasSuffix(n, ".min.js") || n == "index.js" || n == main ||
				strings.Contains(n, "config") || strings.HasPrefix(n, ".") || toolingFile(n) {
				continue
			}
			out = append(out, strings.TrimSuffix(n, ".js"))
		}
	}
	sort.Strings(out)
	if len(out) > 80 {
		out = out[:80]
	}
	return out
}

// toolingFile: top-level files that are build/test tooling, never imported by app code
// (expo config plugins, jest setup/mocks, metro/babel plugins, CLI scripts).
func toolingFile(n string) bool {
	l := strings.ToLower(strings.TrimSuffix(n, ".js"))
	switch l {
	case "app.plugin", "metro", "babel", "cli", "scripts", "postinstall", "react-native.config":
		return true
	}
	for _, p := range []string{"jest", "mock", "babel-plugin", "metro-", "eslint", "webpack", "rollup"} {
		if strings.Contains(l, p) {
			return true
		}
	}
	return false
}
