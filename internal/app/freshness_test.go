package app

import (
	"strings"
	"testing"
	"time"
)

func TestContinuationPreservesFreshness(t *testing.T) {
	for _, flags := range [][]string{{"--max-age=24h", "--offline"}, {"--fresh"}} {
		o, err := parseOptions(flags)
		if err != nil {
			t.Fatal(err)
		}
		next := continuation("CIV 1240", "cursor", o)
		if o.maxAge > 0 && !strings.Contains(next, "--max-age 24h0m0s") {
			t.Fatal(next)
		}
		if o.offline && !strings.Contains(next, "--offline") {
			t.Fatal(next)
		}
		if o.fresh && !strings.Contains(next, "--fresh") {
			t.Fatal(next)
		}
	}
}

func TestMaxAgeOptions(t *testing.T) {
	for _, args := range [][]string{{"--max-age", "24h"}, {"--offline", "--max-age=24h"}} {
		o, err := parseOptions(args)
		if err != nil || o.maxAge != 24*time.Hour {
			t.Fatalf("%v: %+v %v", args, o, err)
		}
	}
	for _, args := range [][]string{{"--max-age"}, {"--max-age=bad"}, {"--max-age=0"}, {"--max-age=-1h"}, {"--fresh", "--max-age=24h"}} {
		if _, err := parseOptions(args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
