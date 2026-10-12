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
	"os"
	"os/exec"
	"path/filepath"
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
}

type Job struct{ Name, Version, RN string }

type Stat struct {
	Job
	OK                                      bool   `json:"ok"`
	Err                                     string `json:"err,omitempty"`
	Skipped                                 string `json:"skipped,omitempty"`
	Install, Metro, Hermes, Hash, Db, Total float64
	Funcs, Kept                             int
	BundleSHA                               string
	CacheHit                                bool
}

func rnNum(rn string) string { // rn074 -> 0.74
	var n int
	fmt.Sscanf(strings.TrimPrefix(rn, "rn"), "%d", &n)
	return fmt.Sprintf("%.2f", float64(n)/100)
}

type key3 struct{ s, c1, c2 string }

type Runner struct {
	cfg       Config
	db        *mongo.Database
	baseline  map[string]map[key3]struct{} // rnNum -> set
	pkgIDs    map[string]primitive.ObjectID
	done      map[string]bool // pkgid|rn
	mu        sync.Mutex
	statsF    *os.File
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
await Metro.runBuild(config, { entry: '%s', platform: 'ios', minify: false, out: '%s' });`

func writeBundleScript(env, entry, out string, reset bool) error {
	return os.WriteFile(filepath.Join(env, "bundle-fast.mjs"), []byte(fmt.Sprintf(bundleScript, reset, entry, out)), 0644)
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
				ContentIR1Hash: h.ContentAR1.Hash, ContentIR2Hash: h.ContentAR2.Hash})
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
		set[key3{h.StructuralHash, h.ContentIR1Hash, h.ContentIR2Hash}] = struct{}{}
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
	r.mu.Lock()
	isDone := r.done[dk]
	r.mu.Unlock()
	if isDone {
		st.OK, st.Skipped = true, "already-in-db"
		return
	}
	t := time.Now()
	if _, err := run(env, 10*time.Minute, "npm", "install", j.Name+"@"+j.Version, "--no-save", "--no-audit", "--no-fund",
		"--ignore-scripts", "--prefer-offline", "--loglevel=error"); err != nil {
		st.Err = "install: " + err.Error()
		return
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
	os.WriteFile(filepath.Join(env, "package_entry_fast.js"), []byte(entry), 0644)
	out := "package_out_fast.js"
	writeBundleScript(env, "package_entry_fast.js", out, false)
	t = time.Now()
	if _, err := run(env, 10*time.Minute, "node", "bundle-fast.mjs"); err != nil {
		st.Err = "metro: " + err.Error()
		return
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
		if _, inBase := base[key3{h.Structural.Hash, h.ContentAR1.Hash, h.ContentAR2.Hash}]; inBase {
			continue
		}
		d := *h
		d.ID, d.PackageID, d.ReactNativeVersion = primitive.NewObjectID(), id, v
		docs = append(docs, d)
	}
	st.Kept = len(docs)
	st.Hash = time.Since(t).Seconds()
	t = time.Now()
	// idempotent: drop partial inserts from an interrupted previous run
	r.db.Collection("hashes").DeleteMany(context.Background(), bson.M{"package_id": id, "react_native_version": v})
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
