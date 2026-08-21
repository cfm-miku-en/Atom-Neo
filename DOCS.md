# Atom-Neo reference

Back to the [README](README.md).

- [The language](#the-language)
- [Builtins](#builtins)
- [Commands](#commands)
- [Projects and packages](#projects-and-packages)
- [The web module](#the-web-module)
- [The proxy](#the-proxy)
- [Speed](#speed)
- [Building](#building)

---

## The language

Everything that produces a value lives in `[ ]`. That one rule covers arithmetic,
conditions, lists, maps and indexing.

### Values

```
var n = 42
var s = "hello"
var b = true
var total = [n * 2 + 1]
```

Strings support `\n`, `\t`, `\"` and `\\`.

### Lists

A bracket with commas is a list. `@` reads an element.

```
var xs = [1, 2, 3]
print([xs @ 0])
push(xs, 4)
print([len(xs)])
print([[1, 2] + [3, 4]])
```

### Maps

A bracket with `key: value` pairs is a map. `@` reads those too.

```
var m = [name: "atom", version: 3]
print([m @ "name"])
put(m, "engine", "Quark")

var empty = [:]
```

Keys print and iterate in sorted order, so runs repeat.

Nested data works as you would expect:

```
var cfg = [host: "localhost", ports: [80, 443]]
print([(cfg @ "ports") @ 1])
```

### Conditions

```
if [n > 10] {
	print("big")
}
else if [n > 3] {
	print("medium")
}
else {
	print("small")
}
```

### Loops

```
while [i < 10] {
	var i = [i + 1]
}

repeat 3 {
	print("tick")
}

each x in xs {
	if [x > 5] {
		break
	}
	print(x)
}
```

`each` over a map walks its keys. `break` and `continue` work in every loop.

### Functions

```
func add(a, b) {
	return [a + b]
}

func fact(n) {
	if [n <= 1] {
		return 1
	}
	return [n * fact(n - 1)]
}

print([fact(10)])
```

Each call gets its own frame, so recursion works and parameters do not leak. A function can
be called from a line above the one that defines it.

Assignment inside a function creates a local. To write an outer variable, say so:

```
global count = 0

func bump() {
	global count = [count + 1]
}
```

Functions are values, which is how a handler reaches the web module:

```
web.handle("/hello", hello)
```

### Operators

| | |
|---|---|
| Arithmetic | `+` `-` `*` `/` `%` |
| Compare | `==` `!=` `?=` `<` `>` `<=` `>=` |
| Logic | `&&` `\|\|` `!` |
| Element | `@` |

`?=` is an alternative spelling of `!=`. `+` also joins strings and lists.

### Comments

```
// to the end of the line
```

---

## Builtins

| | |
|---|---|
| Lists | `push` `pop` `first` `last` `reverse` `range` `contains` `len` |
| Maps | `dict` `keys` `values` `has` `put` `del` `len` |
| Strings | `upper` `lower` `trim` `split` `join` `contains` `len` `lines` |
| Math | `floor` `ceil` `round` `abs` `sqrt` `min` `max` `random` `randint` |
| Convert | `str` `num` `tojson` `fromjson` |
| Files | `read` `write` `append` `exists` `remove` |
| Input | `input` |
| Core | `print` `wait` `exit` |

`push` appends to a list; `append` appends to a file.

`lines(text)` splits on newlines, which is handy with `read`:

```
each row in [lines(read("data.txt"))] {
	print([upper(row)])
}
```

`tojson` and `fromjson` round-trip lists and maps:

```
var data = [name: "atom", tags: ["fast", "small"]]
var back = [fromjson(tojson(data))]
print([back @ "name"])
```

---

## Commands

```
atom run [file.atom]     run the project here, or one file
atom run --watch         restart when files change
atom repl                try statements as you type them
atom fmt [file]          tidy indentation, --check to only report
atom install <pkg>       install a package folder or zip
atom benchmark           measure this build
atom proxy --to ...      reverse proxy, optionally with https
atom <file.atom>         run one file
```

`atom benchmark` takes `--compare` (against local python), `--json`, and
`--ask <question>` where the question is `FasterThanPython`, `Speed` or `Slowest`.

`atom fmt --check` exits non-zero when something needs formatting, so it fits in CI.

---

## Projects and packages

A project is a folder with an `atom.json`:

```json
{
  "name": "mysite",
  "version": "0.1.0",
  "main": "main.atom"
}
```

`atom run` with no arguments runs `main`.

A package is a folder with an `atom.json` in it, or that folder zipped up. Both install the
same way:

```bash
atom install packages/web
atom install web.zip
```

That drops it into `atom_modules/`, and `import web` makes it available. Packages either
carry `.atom` source or bind to a module compiled into the interpreter.

Names are normalised the way PyPI learned to: case is ignored and `-`, `_` and `.` all
collapse together, so two packages cannot differ only by punctuation.

---

## The web module

```
import web

func hello(path) {
	return "<h1>hello from atom</h1>"
}

web.static("./site")
web.notfound("404.html")
web.route("/", "index.html")
web.handle("/hello", hello)
web.log(true)
web.listen(9143)
```

| | |
|---|---|
| `web.static(dir)` | serve a folder, with clean urls and no directory listings |
| `web.route(path, file)` | map a url to a file |
| `web.text(path, body)` | serve a literal string |
| `web.handle(path, fn)` | run an Atom function and serve what it returns |
| `web.notfound(file)` | custom 404 page |
| `web.host(addr)` | bind address, e.g. `127.0.0.1` behind a proxy |
| `web.log(true)` | log requests |
| `web.listen(port)` | start serving |

Inside a handler, `web.path()`, `web.method()` and `web.query(name)` describe the request:

```
func greet(path) {
	var who = [web.query("who")]
	if [who == ""] {
		return "<p>nobody</p>"
	}
	return ["<h1>hello " + who + "</h1>"]
}
```

Clean urls mean `/about` serves `about.html` when no exact match exists.

Some details it does not leave to chance: mime types are registered explicitly rather than
read from the Windows registry, where javascript is regularly served as plain text;
directory listings are off, so a folder without an index is not browsable; and the server
carries read, write and idle timeouts, without which a slow client can hold a connection
open indefinitely.

Handlers run one at a time. The interpreter keeps its scope in package state, so requests
are serialised rather than allowed to corrupt each other's variables. Static files are not
affected.

---

## The proxy

```bash
atom proxy --to localhost:9143
atom proxy --route example.com=localhost:9143 --route api.example.com=localhost:9000
atom proxy --route example.com=localhost:9143 --tls --accept-tos
```

| | |
|---|---|
| `--to host:port` | where to send anything unmatched |
| `--route host=target` | send one domain to its own target, repeatable |
| `--listen port` | plain http port, default 9144 |
| `--tls` | https on 443, with a redirect from 80 |
| `--accept-tos` | confirm Let's Encrypt's subscriber agreement |
| `--certs dir` | where to cache certificates, default `atom-certs` |

`--tls` requires `--route`, so certificates are only ever requested for domains you named
rather than anything that resolves to the box. It also requires `--accept-tos`, because
issuing a certificate accepts [Let's Encrypt's agreement](https://letsencrypt.org/repository/)
and that is not something to do on your behalf quietly.

The proxy sets `X-Forwarded-Host` and `X-Forwarded-Proto`, returns 502 when the upstream is
down, and carries the same timeouts as the web module.

---

## Speed

`atom benchmark --compare` runs the suite against whichever Python is on the same machine.
Iteration counts are calibrated per language to hit a target duration, and each case is
timed several times keeping the fastest. Published numbers from other people's hardware
would not mean anything, so it measures both here.

One run against CPython 3.14.6:

```
benchmark          atom       python      ratio
-----------------------------------------------
loop            25.0M/s      18.7M/s      1.33x
arith           11.1M/s       8.5M/s      1.30x
branch          14.3M/s      13.1M/s      1.09x
```

Your numbers will differ. Run it yourself rather than trusting these.

---

## Building

```bash
go build -o atom ./src
go test ./src/...
```

Needs Go 1.26.5+. The binary is statically linked, so it needs nothing installed on the
machine it runs on. For a server:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o atom ./src
```

Tests are `.atom` programs in `src/parser/testdata` checked against their expected output,
plus unit tests for the engine.
