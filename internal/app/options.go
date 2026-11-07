package app

import (
	"flag"
	"io"
	"strings"
	"time"

	"github.com/bpineau/legidor/internal/model"
	"github.com/bpineau/legidor/internal/reference"
)

type options struct {
	json, help, fresh, offline, full, parts, parseOnly, diff bool
	date, part, grep, from, to, since, corpus, cursor, env   string
	limit                                                    int
	maxAge                                                   time.Duration
	args                                                     []string
	supplied                                                 map[string]bool
}

func parseOptions(args []string) (options, error) {
	o := options{supplied: map[string]bool{}}
	fs := flag.NewFlagSet("legidor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&o.json, "json", false, "")
	fs.BoolVar(&o.help, "help", false, "")
	fs.BoolVar(&o.help, "h", false, "")
	fs.BoolVar(&o.fresh, "fresh", false, "")
	fs.DurationVar(&o.maxAge, "max-age", 0, "")
	fs.BoolVar(&o.offline, "offline", false, "")
	fs.BoolVar(&o.full, "full", false, "")
	fs.BoolVar(&o.parts, "parts", false, "")
	fs.BoolVar(&o.parseOnly, "parse-only", false, "")
	fs.BoolVar(&o.diff, "diff", false, "")
	fs.StringVar(&o.date, "date", "", "")
	fs.StringVar(&o.part, "part", "", "")
	fs.StringVar(&o.grep, "grep", "", "")
	fs.StringVar(&o.from, "from", "", "")
	fs.StringVar(&o.to, "to", "", "")
	fs.StringVar(&o.since, "since", "", "")
	fs.StringVar(&o.corpus, "in", "loda", "")
	fs.StringVar(&o.cursor, "cursor", "", "")
	fs.StringVar(&o.env, "env", "production", "")
	fs.IntVar(&o.limit, "limit", 10, "")
	var flags, pos []string
	stop := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			stop = true
			continue
		}
		if stop || !strings.HasPrefix(a, "-") || a == "-" {
			pos = append(pos, a)
			continue
		}
		name, _, eq := strings.Cut(strings.TrimLeft(a, "-"), "=")
		f := fs.Lookup(name)
		if f == nil {
			return o, model.Err("USAGE", 2, "unknown option %s", a)
		}
		flags = append(flags, a)
		o.supplied[name] = true
		bf, isBool := f.Value.(interface{ IsBoolFlag() bool })
		if !eq && (!isBool || !bf.IsBoolFlag()) {
			i++
			if i >= len(args) {
				return o, model.Err("USAGE", 2, "option %s requires a value", a)
			}
			flags = append(flags, args[i])
		}
	}
	if e := fs.Parse(flags); e != nil {
		return o, model.Err("USAGE", 2, "invalid option value")
	}
	o.args = pos
	if o.supplied["max-age"] && (o.maxAge <= 0 || o.fresh) {
		return o, model.Err("USAGE", 2, "--max-age requires a positive duration and cannot be combined with --fresh")
	}
	if o.fresh && o.offline {
		return o, model.Err("CONFLICT", 2, "--fresh and --offline cannot be combined")
	}
	for _, d := range []string{o.date, o.from, o.to, o.since} {
		if d != "" && !reference.ValidDate(d) {
			return o, model.Err("USAGE", 2, "dates must be YYYY-MM-DD")
		}
	}
	if o.limit < 1 || o.limit > 50 {
		return o, model.Err("USAGE", 2, "limit must be 1..50")
	}
	return o, nil
}
