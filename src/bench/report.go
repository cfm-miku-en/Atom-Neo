package bench

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func rate(v float64) string {
	switch {
	case v >= 1000000:
		return fmt.Sprintf("%.1fM/s", v/1000000)
	case v >= 1000:
		return fmt.Sprintf("%.0fk/s", v/1000)
	default:
		return fmt.Sprintf("%.0f/s", v)
	}
}

func Table(results []Result) {
	compared := false
	for _, r := range results {
		if r.Compared {
			compared = true
		}
	}

	if compared {
		fmt.Printf("%-10s %12s %12s %10s\n", "benchmark", "atom", "python", "ratio")
		fmt.Println(strings.Repeat("-", 47))
		for _, r := range results {
			verdict := fmt.Sprintf("%.2fx", r.Ratio())
			if r.Ratio() < 1 {
				verdict = fmt.Sprintf("%.0fx slower", 1/r.Ratio())
			}
			fmt.Printf("%-10s %12s %12s %10s\n", r.Name, rate(r.AtomPerSec()), rate(r.PythonPerSec()), verdict)
		}
		if v := PythonVersion(); v != "" {
			fmt.Printf("\ncompared against %s on this machine\n", v)
		}
		return
	}

	fmt.Printf("%-10s %12s %14s\n", "benchmark", "ops/sec", "per op")
	fmt.Println(strings.Repeat("-", 38))
	for _, r := range results {
		per := r.AtomTime.Nanoseconds() / int64(r.AtomOps)
		fmt.Printf("%-10s %12s %11dns\n", r.Name, rate(r.AtomPerSec()), per)
	}
}

func JSON(results []Result) {
	type entry struct {
		Name         string  `json:"name"`
		Ops          int     `json:"ops"`
		AtomSeconds  float64 `json:"atom_seconds"`
		AtomPerSec   float64 `json:"atom_ops_per_sec"`
		PythonPerSec float64 `json:"python_ops_per_sec,omitempty"`
		Ratio        float64 `json:"ratio_atom_over_python,omitempty"`
	}

	out := make([]entry, 0, len(results))
	for _, r := range results {
		e := entry{
			Name:        r.Name,
			Ops:         r.AtomOps,
			AtomSeconds: r.AtomTime.Seconds(),
			AtomPerSec:  r.AtomPerSec(),
		}
		if r.Compared {
			e.PythonPerSec = r.PythonPerSec()
			e.Ratio = r.Ratio()
		}
		out = append(out, e)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(out)
}

var Questions = []string{"FasterThanPython", "Speed", "Slowest"}

func Ask(question string, results []Result) {
	switch strings.ToLower(question) {
	case "fasterthanpython":
		total, wins := 0, 0
		for _, r := range results {
			if !r.Compared {
				continue
			}
			total++
			if r.Ratio() > 1 {
				wins++
			}
		}
		if total == 0 {
			fmt.Println("unknown - no python on this machine to compare against")
			return
		}
		if wins == total {
			fmt.Printf("yes - atom won all %d benchmarks\n", total)
			return
		}
		if wins == 0 {
			worst := results[0]
			for _, r := range results {
				if r.Compared && r.Ratio() < worst.Ratio() {
					worst = r
				}
			}
			fmt.Printf("no - python won all %d, by up to %.0fx (%s)\n", total, 1/worst.Ratio(), worst.Name)
			return
		}
		fmt.Printf("partly - atom won %d of %d\n", wins, total)

	case "speed":
		for _, r := range results {
			fmt.Printf("%s %s\n", r.Name, rate(r.AtomPerSec()))
		}

	case "slowest":
		slowest := results[0]
		for _, r := range results {
			if r.AtomPerSec() < slowest.AtomPerSec() {
				slowest = r
			}
		}
		fmt.Printf("%s at %s\n", slowest.Name, rate(slowest.AtomPerSec()))

	default:
		fmt.Printf("unknown question '%s', try: %s\n", question, strings.Join(Questions, ", "))
	}
}
