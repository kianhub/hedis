package pipeline

import (
	"errors"
	"testing"

	"hermes-decompiler/pkg/database/models"
)

func TestMaxPackageNSNCCoverage(t *testing.T) {
	// Healthy: one package shares ~95% of its keys with the app.
	healthyKeys := map[nkey]struct{}{}
	A := map[nkey]struct{}{}
	for i := 0; i < 100; i++ {
		k := nkey{s: "hS", c: "hC" + string(rune('A'+i%20)) + string(rune('0'+i/20))}
		// make unique keys
		k = nkey{s: "hs" + itoa(i), c: "hc" + itoa(i)}
		healthyKeys[k] = struct{}{}
		if i < 95 {
			A[k] = struct{}{}
		}
	}
	// Unsupported-like: packages share only ~10% with the app.
	unsupKeys := map[nkey]struct{}{}
	for i := 0; i < 100; i++ {
		k := nkey{s: "us" + itoa(i), c: "uc" + itoa(i)}
		unsupKeys[k] = struct{}{}
		if i < 10 {
			A[k] = struct{}{}
		}
	}
	// Tiny package (< minKeys) with 100% overlap must not dominate.
	tiny := map[nkey]struct{}{{s: "t0", c: "t0"}: {}, {s: "t1", c: "t1"}: {}}
	for k := range tiny {
		A[k] = struct{}{}
	}

	pvsHealthy := []*pvKeys{
		{pkg: &models.PackageWithVulnerabilitiesModel{PackageName: "zod", PackageVersion: "3.23.8"}, keys: healthyKeys},
		{pkg: &models.PackageWithVulnerabilitiesModel{PackageName: "tiny", PackageVersion: "1.0.0"}, keys: tiny},
	}
	cov, best := MaxPackageNSNCCoverage(A, pvsHealthy, 20)
	if cov < 0.90 || best != "zod@3.23.8" {
		t.Fatalf("healthy: cov=%v best=%q want ~0.95 zod@3.23.8", cov, best)
	}
	if cov < DefaultUnsupportedMinCov {
		t.Fatalf("healthy cov %v should pass DefaultUnsupportedMinCov %v", cov, DefaultUnsupportedMinCov)
	}

	pvsUnsup := []*pvKeys{
		{pkg: &models.PackageWithVulnerabilitiesModel{PackageName: "axios", PackageVersion: "1.7.9"}, keys: unsupKeys},
		{pkg: &models.PackageWithVulnerabilitiesModel{PackageName: "zod", PackageVersion: "3.23.8"}, keys: map[nkey]struct{}{
			{s: "z0", c: "z0"}: {}, {s: "z1", c: "z1"}: {}, // will add more below
		}},
	}
	// Expand zod-unsup to 100 keys with 12% overlap (mirrors rn085 diag max ~0.126).
	zkeys := map[nkey]struct{}{}
	for i := 0; i < 100; i++ {
		k := nkey{s: "zs" + itoa(i), c: "zc" + itoa(i)}
		zkeys[k] = struct{}{}
		if i < 12 {
			A[k] = struct{}{}
		}
	}
	pvsUnsup[1].keys = zkeys
	cov2, best2 := MaxPackageNSNCCoverage(A, pvsUnsup, 20)
	if cov2 > 0.15 || best2 != "zod@3.23.8" {
		t.Fatalf("unsupported-like: cov=%v best=%q want ~0.12 zod@3.23.8", cov2, best2)
	}
	if cov2 >= DefaultUnsupportedMinCov {
		t.Fatalf("unsupported cov %v should fail DefaultUnsupportedMinCov %v", cov2, DefaultUnsupportedMinCov)
	}
}

func TestUnsupportedBytecodeError(t *testing.T) {
	err := &UnsupportedBytecodeError{Coverage: 0.126, Min: DefaultUnsupportedMinCov, BestPkg: "zod@3.23.8"}
	if !errors.Is(err, ErrUnsupportedBytecode) {
		t.Fatal("errors.Is should match ErrUnsupportedBytecode")
	}
	msg := err.Error()
	if !contains(msg, "React Native or bytecode version not supported") {
		t.Fatalf("message missing clear error text: %q", msg)
	}
	if !contains(msg, "0.126") || !contains(msg, "0.400") {
		t.Fatalf("message should include coverage numbers: %q", msg)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [16]byte
	n := len(b)
	for i > 0 {
		n--
		b[n] = byte('0' + i%10)
		i /= 10
	}
	return string(b[n:])
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}
