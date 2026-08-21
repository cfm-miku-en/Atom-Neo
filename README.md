<div align="center">
  <img width="640" height=auto alt="Atom" src="https://github.com/user-attachments/assets/04a14b8e-8dfb-4dd5-bd30-a76a5790de93">
</div>

![stars](https://img.shields.io/github/stars/cfm-miku-en/Atom3?style=social)
![engine](https://img.shields.io/badge/engine-Quark-7ee787)
![deps](https://img.shields.io/badge/language%20deps-0-blue)
![beginner](https://img.shields.io/badge/beginner-friendly-orange)

# Atom-Neo

A fork of [Atom3](https://github.com/WawaDevX/Atom3) by WawaDev, aimed at being fast and
featureful. It runs on **Quark**, a hand-written expression engine and tree-walking
interpreter with no third-party parsing libraries.

> [!WARNING]
> Not production ready. Things will move around.

---

## Install

**Windows**

```powershell
powershell -ExecutionPolicy Bypass -File install.ps1
```

Builds `atom.exe`, puts it in `%LOCALAPPDATA%\Atom-Neo\bin`, and adds that to your user
PATH. No administrator rights. `uninstall.ps1` reverses it.

**From source** (needs Go 1.26.5+)

```bash
go build -o atom ./src
go test ./src/...
```

The binary is statically linked, so a built `atom` needs nothing installed on the machine
it runs on. Cross-compile for a server with:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o atom ./src
```

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

### Conditions and loops

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

### Operators

| | |
|---|---|
| Arithmetic | `+` `-` `*` `/` `%` |
| Compare | `==` `!=` `?=` `<` `>` `<=` `>=` |
| Logic | `&&` `\|\|` `!` |
| Element | `@` |

`?=` is an alternative spelling of `!=`. `+` also joins strings and lists.

### Builtins

| | |
|---|---|
| Lists | `push` `pop` `first` `last` `reverse` `range` `contains` `len` |
| Maps | `dict` `keys` `values` `has` `put` `del` `len` |
| Strings | `upper` `lower` `trim` `split` `join` `contains` `len` `lines` |
| Math | `floor` `ceil` `round` `abs` `sqrt` `min` `max` `random` `randint` |
| Convert | `str` `num` `tojson` `fromjson` |
| Files | `read` `write` `append` `exists` `remove` |
| Input | `input` |

`push` appends to a list; `append` appends to a file.

---

## Commands

```
atom run [file.atom]     run the project here, or one file
atom run --watch         restart when files change
atom repl                try statements as you type them
atom fmt [file]          tidy indentation, --check to only report
atom install <pkg.zip>   install a package into atom_modules
atom benchmark           measure this build
atom proxy --to ...      reverse proxy, optionally with https
atom <file.atom>         run one file
```

A project is a folder with an `atom.json`:

```json
{
  "name": "mysite",
  "version": "0.1.0",
  "main": "main.atom"
}
```

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

Inside a handler, `web.path()`, `web.method()` and `web.query(name)` describe the request.

Mime types are registered explicitly rather than read from the Windows registry, directory
listings are off, and the server carries read, write and idle timeouts.

Handlers run one at a time. The interpreter keeps its scope in package state, so requests
are serialised rather than allowed to corrupt each other.

---

## The proxy

```bash
atom proxy --to localhost:9143
atom proxy --route example.com=localhost:9143 --route api.example.com=localhost:9000
atom proxy --route example.com=localhost:9143 --tls --accept-tos
```

`--tls` serves https on 443 with certificates from Let's Encrypt and redirects port 80. It
requires `--route`, so certificates are only requested for domains you named, and
`--accept-tos`, because issuing one accepts their subscriber agreement.

---

## Speed

`atom benchmark --compare` runs the suite against whichever Python is on the same machine,
calibrating iteration counts per language and keeping the fastest of several runs.
Published numbers from other people's hardware would not mean anything, so it measures both
here.

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

## Credit

Atom3 is by [WawaDev](https://github.com/WawaDevX) (formerly spacecat) and the Atom3 team.
This fork keeps their language and rewrites the machinery under it.
