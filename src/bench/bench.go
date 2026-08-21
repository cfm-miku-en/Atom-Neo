package bench

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"Atom3/src/parser"
)

// target is how long each measured run should take. Iteration counts are
// calibrated per language to hit it, so both are timed in a range where the
// measurement is well clear of startup and scheduler noise.
const target = 350 * time.Millisecond

const probe = 500

type Case struct {
	Name   string
	Atom   func(n int) string
	Python func(n int) string
}

type Result struct {
	Name       string
	AtomOps    int
	AtomTime   time.Duration
	PythonOps  int
	PythonTime time.Duration
	Compared   bool
}

func perSec(ops int, d time.Duration) float64 {
	if d <= 0 {
		return 0
	}
	return float64(ops) / d.Seconds()
}

func (r Result) AtomPerSec() float64   { return perSec(r.AtomOps, r.AtomTime) }
func (r Result) PythonPerSec() float64 { return perSec(r.PythonOps, r.PythonTime) }

// Ratio is how many times faster Atom is than Python; below 1 means slower.
func (r Result) Ratio() float64 {
	a, p := r.AtomPerSec(), r.PythonPerSec()
	if a <= 0 || p <= 0 {
		return 0
	}
	return a / p
}

var cases = []Case{
	{
		Name: "loop",
		Atom: func(n int) string {
			return fmt.Sprintf("var i = 0\nwhile [i < %d] {\nvar i = [i + 1]\n}", n)
		},
		Python: func(n int) string {
			return fmt.Sprintf("i = 0\nwhile i < %d:\n    i = i + 1", n)
		},
	},
	{
		Name: "arith",
		Atom: func(n int) string {
			return fmt.Sprintf("var i = 0\nvar t = 0\nwhile [i < %d] {\nvar t = [t + i * 2]\nvar i = [i + 1]\n}", n)
		},
		Python: func(n int) string {
			return fmt.Sprintf("i = 0\nt = 0\nwhile i < %d:\n    t = t + i * 2\n    i = i + 1", n)
		},
	},
	{
		Name: "branch",
		Atom: func(n int) string {
			return fmt.Sprintf("var i = 0\nwhile [i < %d] {\nif [i > 5] {\nvar i = [i + 1]\n}\nelse {\nvar i = [i + 1]\n}\n}", n)
		},
		Python: func(n int) string {
			return fmt.Sprintf("i = 0\nwhile i < %d:\n    if i > 5:\n        i = i + 1\n    else:\n        i = i + 1", n)
		},
	},
}

func runAtom(src string) time.Duration {
	parser.ResetVariables()

	start := time.Now()
	parser.RunSource(src)
	return time.Since(start)
}

func scale(n int, took time.Duration) int {
	if took <= 0 {
		return n * 100
	}
	next := int(float64(n) * (float64(target) / float64(took)))
	if next < n {
		next = n
	}
	if next > 20000000 {
		next = 20000000
	}
	return next
}

// Noise only ever adds time, so the fastest of several runs is the closest
// estimate of the real cost.
const repeats = 5

func measureAtom(c Case) (int, time.Duration) {
	n := scale(probe, runAtom(c.Atom(probe)))

	best := time.Duration(0)
	for i := 0; i < repeats; i++ {
		if d := runAtom(c.Atom(n)); best == 0 || d < best {
			best = d
		}
	}
	return n, best
}

func pythonCmd() string {
	for _, name := range []string{"python3", "python"} {
		if _, err := exec.LookPath(name); err == nil {
			return name
		}
	}
	return ""
}

func timePython(bin, src, dir string) time.Duration {
	path := filepath.Join(dir, "atom_bench.py")
	if os.WriteFile(path, []byte(src+"\n"), 0o644) != nil {
		return 0
	}
	defer os.Remove(path)

	start := time.Now()
	if exec.Command(bin, path).Run() != nil {
		return 0
	}
	return time.Since(start)
}

// baseline is interpreter startup, subtracted so Python's figure is execution
// time only and therefore comparable with Atom's in-process timing.
func baseline(bin, dir string) time.Duration {
	best := time.Duration(0)
	for i := 0; i < 3; i++ {
		d := timePython(bin, "pass", dir)
		if d <= 0 {
			return 0
		}
		if best == 0 || d < best {
			best = d
		}
	}
	return best
}

func measurePython(bin string, c Case, dir string, base time.Duration) (int, time.Duration) {
	net := func(n int) time.Duration {
		d := timePython(bin, c.Python(n), dir)
		if d > base {
			return d - base
		}
		return 0
	}

	n := probe
	took := net(n)
	for i := 0; i < 4 && took < 50*time.Millisecond; i++ {
		n *= 20
		took = net(n)
	}

	n = scale(n, took)

	best := time.Duration(0)
	for i := 0; i < 3; i++ {
		if d := net(n); d > 0 && (best == 0 || d < best) {
			best = d
		}
	}
	return n, best
}

func Run(compare bool) []Result {
	bin := ""
	base := time.Duration(0)
	dir := os.TempDir()

	if compare {
		if bin = pythonCmd(); bin != "" {
			base = baseline(bin, dir)
		}
	}

	results := make([]Result, 0, len(cases))
	for _, c := range cases {
		r := Result{Name: c.Name}
		r.AtomOps, r.AtomTime = measureAtom(c)

		if bin != "" {
			r.PythonOps, r.PythonTime = measurePython(bin, c, dir, base)
			r.Compared = r.PythonTime > 0
		}
		results = append(results, r)
	}
	return results
}

func PythonVersion() string {
	bin := pythonCmd()
	if bin == "" {
		return ""
	}
	out, err := exec.Command(bin, "--version").CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
