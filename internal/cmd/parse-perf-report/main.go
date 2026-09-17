// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// parse-perf-report converts the CSV output of "benchstat" into the markdown
// body of the performance results PR comment.
//
// It reads perf-report.csv and writes perf-report.md. Only the metrics that are
// meaningful to compare across runs are reported: sec/op, B/op and allocs/op.
// The custom ops_per_second_{min,max,med} metrics that the benchmarks report for
// the Evergreen performance monitoring upload are deliberately dropped, because
// they are order statistics derived from single-iteration wall times and are far
// too noisy to compare.
//
// Each metric is gated according to what it actually measures:
//
//   - sec/op is reported when benchstat's Mann-Whitney U test finds the
//     difference significant (benchstat renders everything else as "~").
//   - B/op and allocs/op are compared exactly when they held perfectly still
//     across all repetitions on both sides, since a difference is then a real
//     deterministic change. Where they did vary -- which includes allocs/op for
//     the benchmarks that talk to a server -- they require both significance
//     and a minimum magnitude.
package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
)

const (
	inputFileName  = "perf-report.csv"
	outputFileName = "perf-report.md"

	secPerOp    = "sec/op"
	bytesPerOp  = "B/op"
	allocsPerOp = "allocs/op"
)

// reportedMetrics are rendered, in this order. Anything else benchstat emits is
// dropped.
var reportedMetrics = []string{secPerOp, bytesPerOp, allocsPerOp}

var metricTitles = map[string]string{
	secPerOp:    "Time per operation",
	bytesPerOp:  "Bytes allocated per operation",
	allocsPerOp: "Allocations per operation",
}

// row is one benchmark's result for a single metric.
type row struct {
	name     string
	base     float64
	baseCI   string
	head     float64
	headCI   string
	delta    string // benchstat's "vs base" column: "~" or e.g. "+3.42%"
	pValue   string
	geomean  bool
	haveVals bool
}

// table is all rows for a single metric.
type table struct {
	metric string
	rows   []row
}

func main() {
	tables, err := parse(inputFileName)
	if err != nil {
		log.Panic(err)
	}

	out, err := os.Create(outputFileName)
	if err != nil {
		log.Panic(err)
	}
	defer out.Close()

	render(out, tables)
}

// parse reads benchstat CSV output. The format is a series of blank-line
// separated blocks, each block being:
//
//	,base,,head,,,
//	,<metric>,CI,<metric>,CI,vs base,P
//	<benchmark name>,<base>,<base CI>,<head>,<head CI>,<delta>,<p=... n=...>
//	...
//	geomean,...
//
// Leading "goos:"/"goarch:"/"pkg:"/"cpu:" configuration lines appear before the
// first block.
func parse(path string) ([]table, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %q: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	// benchstat blocks have a varying number of fields, and the configuration
	// preamble has one field per line.
	r.FieldsPerRecord = -1

	var (
		tables  []table
		current *table
	)

	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("failed to read %q: %w", path, err)
		}

		// The "base"/"head" header line starts a new block; the following line
		// names the metric.
		if len(rec) >= 2 && rec[0] == "" && rec[1] == "base" {
			current = nil

			continue
		}

		// Metric header line.
		if len(rec) >= 3 && rec[0] == "" && rec[2] == "CI" {
			tables = append(tables, table{metric: rec[1]})
			current = &tables[len(tables)-1]

			continue
		}

		if current == nil || len(rec) < 6 {
			continue
		}

		rw := row{
			name:    strings.TrimSpace(rec[0]),
			baseCI:  rec[2],
			headCI:  rec[4],
			delta:   strings.TrimSpace(rec[5]),
			geomean: strings.TrimSpace(rec[0]) == "geomean",
		}

		if len(rec) >= 7 {
			rw.pValue = strings.TrimSpace(rec[6])
		}

		base, errBase := strconv.ParseFloat(rec[1], 64)
		head, errHead := strconv.ParseFloat(rec[3], 64)

		if errBase == nil && errHead == nil {
			rw.base, rw.head, rw.haveVals = base, head, true
		}

		current.rows = append(current.rows, rw)
	}

	return tables, nil
}

func render(w io.Writer, tables []table) {
	byMetric := make(map[string]table, len(tables))
	for _, t := range tables {
		byMetric[t.metric] = t
	}

	var body strings.Builder

	timeRegressions, allocChanges := 0, 0

	for _, metric := range reportedMetrics {
		t, ok := byMetric[metric]
		if !ok {
			continue
		}

		title := metricTitles[metric]

		fmt.Fprintf(&body, "\n### %s (`%s`)\n\n", title, metric)
		fmt.Fprint(&body, "| Benchmark | Base | Head | Change | Significance |\n")
		fmt.Fprint(&body, "| --- | --- | --- | --- | --- |\n")

		for _, rw := range t.rows {
			if rw.geomean {
				continue
			}

			change, sig := "~", "no significant difference"

			switch {
			case isAllocMetric(metric) && rw.haveVals:
				// Allocation metrics are only worth comparing exactly when they
				// actually held still. For the BSON benchmarks both sides are
				// perfectly stable, so any difference is a real, deterministic
				// change and is reported without a significance test. For the
				// benchmarks that talk to a server they are not stable -- even
				// allocs/op has a fractional median, meaning the count itself
				// varies between runs -- so fall back to requiring both
				// significance and a minimum magnitude.
				pct := pctChange(rw.base, rw.head)

				switch {
				case rw.base == rw.head:
					change = "no change"
					sig = "identical"
				case rw.delta != "" && rw.delta != "~" && math.Abs(pct) >= allocTolerancePct:
					change = formatPct(pct)
					sig = "**changed**"
					allocChanges++
				default:
					sig = rw.pValue
				}
			default:
				// Trust benchstat's significance test for timing.
				if rw.delta != "" && rw.delta != "~" {
					change = rw.delta
					sig = rw.pValue
					timeRegressions++
				} else if rw.pValue != "" {
					sig = rw.pValue
				}
			}

			fmt.Fprintf(&body, "| `%s` | %s | %s | %s | %s |\n",
				rw.name,
				formatValue(metric, rw.base, rw.baseCI, rw.haveVals),
				formatValue(metric, rw.head, rw.headCI, rw.haveVals),
				change,
				sig,
			)
		}

		for _, rw := range t.rows {
			if rw.geomean && rw.delta != "" {
				fmt.Fprintf(&body, "\nGeomean: %s\n", rw.delta)
			}
		}
	}

	fmt.Fprint(w, "## 🧪 Performance Results\n\n")

	if os.Getenv("PERF_HARNESS_CHANGED") == "true" {
		fmt.Fprint(w, "> [!WARNING]\n")
		fmt.Fprint(w, "> This PR modifies `internal/cmd/benchmark/`. The base and head\n")
		fmt.Fprint(w, "> revisions were measured with their own benchmark code, so the\n")
		fmt.Fprint(w, "> numbers below may not be directly comparable.\n\n")
	}

	fmt.Fprintf(w, "%s\n\n", summary(timeRegressions, allocChanges))

	fmt.Fprint(w, "<details open>\n<summary>Details</summary>\n")
	fmt.Fprint(w, body.String())
	fmt.Fprint(w, "\n</details>\n\n")

	fmt.Fprint(w, "<details>\n<summary>How to read this</summary>\n\n")
	fmt.Fprint(w, "Both revisions were benchmarked on this machine in this task, with runs\n")
	fmt.Fprint(w, "interleaved to cancel out drift. `Base` and `Head` are medians across\n")
	fmt.Fprint(w, "repetitions, with a 95% confidence interval.\n\n")
	fmt.Fprint(w, "- **Time** is compared with [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)'s\n")
	fmt.Fprint(w, "  Mann-Whitney U test. `~` means the difference is indistinguishable from\n")
	fmt.Fprint(w, "  noise. A percentage is shown only when `p < 0.05`.\n")
	fmt.Fprint(w, "- **Allocations** (`B/op`, `allocs/op`) are compared exactly when they held\n")
	fmt.Fprint(w, "  completely still across every repetition, since a difference is then a\n")
	fmt.Fprint(w, "  real deterministic change rather than noise. This is the clearest signal\n")
	fmt.Fprint(w, "  that a change affected performance.\n")
	fmt.Fprintf(w, "- Where allocations did vary between repetitions, they are reported only\n"+
		"  when the difference is both significant and at least ±%.1f%%.\n\n",
		allocTolerancePct)
	fmt.Fprint(w, "To reproduce locally:\n\n")
	fmt.Fprint(w, "```\nBASE_SHA=<base> HEAD_SHA=<head> task perf-pr-report\n```\n")
	fmt.Fprint(w, "\n</details>\n")
}

func summary(timeRegressions, allocChanges int) string {
	if timeRegressions == 0 && allocChanges == 0 {
		return "No statistically significant change in timing, and no change in allocations."
	}

	parts := make([]string, 0, 2)

	if timeRegressions > 0 {
		parts = append(parts, fmt.Sprintf("%s with a significant timing difference",
			plural(timeRegressions, "benchmark")))
	}

	if allocChanges > 0 {
		parts = append(parts, fmt.Sprintf("%s with an allocation change",
			plural(allocChanges, "measurement")))
	}

	return "Found " + strings.Join(parts, " and ") + "."
}

func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}

	return fmt.Sprintf("%d %ss", n, noun)
}

// allocTolerancePct is the minimum magnitude of an allocation difference worth
// reporting for the benchmarks whose allocation counts are not stable run to
// run. It is applied on top of benchstat's significance test, so that neither
// run-to-run noise nor a statistically real but negligible difference is
// reported as a change.
const allocTolerancePct = 0.5

func isAllocMetric(metric string) bool {
	return metric == bytesPerOp || metric == allocsPerOp
}

func pctChange(base, head float64) float64 {
	if base == 0 {
		return 0
	}

	return (head - base) / base * 100
}

func formatPct(pct float64) string {
	return fmt.Sprintf("%+.2f%%", pct)
}

// formatValue renders a metric value in human-readable units, with benchstat's
// confidence interval appended.
func formatValue(metric string, v float64, ci string, haveVals bool) string {
	if !haveVals {
		return "n/a"
	}

	var s string

	switch metric {
	case secPerOp:
		s = formatDuration(v)
	case bytesPerOp:
		s = formatBytes(v)
	default:
		s = strconv.FormatFloat(v, 'f', -1, 64)
	}

	if ci != "" && ci != "0%" {
		s += " ± " + ci
	}

	return s
}

func formatDuration(sec float64) string {
	switch {
	case sec >= 1:
		return fmt.Sprintf("%.3f s", sec)
	case sec >= 1e-3:
		return fmt.Sprintf("%.3f ms", sec*1e3)
	case sec >= 1e-6:
		return fmt.Sprintf("%.3f µs", sec*1e6)
	default:
		return fmt.Sprintf("%.3f ns", sec*1e9)
	}
}

func formatBytes(b float64) string {
	const unit = 1024

	switch {
	case b >= unit*unit:
		return fmt.Sprintf("%.2f MiB", b/unit/unit)
	case b >= unit:
		return fmt.Sprintf("%.2f KiB", b/unit)
	default:
		return fmt.Sprintf("%.0f B", b)
	}
}
