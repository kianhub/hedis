package cmd

import (
	"encoding/json"
	"hermes-decompiler/pkg/fastpipe"
	"hermes-decompiler/pkg/utils"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	fpTargets, fpRNs, fpEnvs, fpCache, fpStats string
	fpWorkers                                  int
)

// fingerprintCmd: parallel, cached package fingerprint DB build (see pkg/fastpipe).
// targets JSON: {"packages": {"<name>": {"all": ["<version>", ...]}}}
var fingerprintCmd = &cobra.Command{
	Use:   "fingerprint",
	Short: "Fast parallel fingerprint DB build for a target list of package versions",
	Run: func(cmd *cobra.Command, args []string) {
		var t struct {
			Packages map[string]struct {
				All []string `json:"all"`
			} `json:"packages"`
		}
		b, err := os.ReadFile(fpTargets)
		if err != nil {
			utils.F.Println(err)
			os.Exit(1)
		}
		if err := json.Unmarshal(b, &t); err != nil {
			utils.F.Println(err)
			os.Exit(1)
		}
		fpEnvs, _ = filepath.Abs(fpEnvs)
		fpCache, _ = filepath.Abs(fpCache)
		rns := strings.Split(fpRNs, ",")
		var jobs []fastpipe.Job
		for _, rn := range rns {
			for n, p := range t.Packages {
				for _, v := range p.All {
					jobs = append(jobs, fastpipe.Job{Name: n, Version: v, RN: rn})
				}
			}
		}
		dbName := os.Getenv("MONGO_DB_NAME")
		if dbName == "" {
			dbName = "clean"
		}
		osh := os.Getenv("OS_HERMES")
		if osh == "" {
			osh = "linux64-bin"
		}
		r, err := fastpipe.New(fastpipe.Config{MongoURI: os.Getenv("MONGO_CONNECTION_STRING"), DBName: dbName,
			EnvsRoot: fpEnvs, RNs: rns, Workers: fpWorkers, CacheDir: fpCache, StatsPath: fpStats, OSHermes: osh})
		if err != nil {
			utils.F.Println(err)
			os.Exit(1)
		}
		utils.L.Printf("fingerprint: %d jobs, %d workers per RN\n", len(jobs), fpWorkers)
		if err := r.Run(jobs); err != nil {
			utils.F.Println(err)
			os.Exit(1)
		}
		utils.S.Println("fingerprint: done")
	},
}

func init() {
	fingerprintCmd.Flags().StringVar(&fpTargets, "targets", "", "targets JSON")
	fingerprintCmd.Flags().StringVar(&fpRNs, "rns", "rn074,rn076", "comma-separated RN env dir names")
	fingerprintCmd.Flags().StringVar(&fpEnvs, "envs-root", "../../pipeline/react-natives", "dir containing rnXXX envs")
	fingerprintCmd.Flags().IntVar(&fpWorkers, "workers", 3, "workers per RN version")
	fingerprintCmd.Flags().StringVar(&fpCache, "cache", ".fpcache", "hbc cache dir")
	fingerprintCmd.Flags().StringVar(&fpStats, "stats", "fingerprint-stats.jsonl", "per-job timing JSONL")
	rootCmd.AddCommand(fingerprintCmd)
}
