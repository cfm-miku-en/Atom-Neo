<div align="center">
  <img width="640" height=auto alt="Atom" src="[https://github.com/user-attachments/assets/04a14b8e-8dfb-4dd5-bd30-a76a5790de93](https://github.com/cfm-miku-en/Atom-Neo/blob/main/image0.png?raw=true)">
</div>

![stars](https://img.shields.io/github/stars/cfm-miku-en/Atom3?style=social)
![engine](https://img.shields.io/badge/engine-Quark-7ee787)
![deps](https://img.shields.io/badge/language%20deps-0-blue)
![beginner](https://img.shields.io/badge/beginner-friendly-orange)

# Atom-Neo

A fork of [Atom3](https://github.com/WawaDevX/Atom3) by WawaDev, built to be fast and to
come with things already in the box: lists, maps, json, a web server, a reverse proxy, a
repl and a formatter.

Runs on **Quark**, a hand-written engine with no third-party parsing libraries.

> [!WARNING]
> Not production ready. Things will move around.

```
var xs = [1, 2, 3]

each x in xs {
	print([x * 2])
}

func hi(name) {
	return ["hello " + name]
}

print([hi("world")])
```

## Install

```powershell
powershell -ExecutionPolicy Bypass -File install.ps1
```

Or build it: `go build -o atom ./src` (Go 1.26.5+).

## Speed

Beats CPython 3.14 on all three built-in benchmarks. Don't take my word for it:

```bash
atom benchmark --compare
```

## Docs

- **[docs/reference.md](docs/reference.md)** — the language, every builtin, the commands, the web module and the proxy
- **[docs/packages.md](docs/packages.md)** — how to make and share a package

## Credit

Atom3 is by [WawaDev](https://github.com/WawaDevX) (formerly spacecat) and the Atom3 team.
This fork keeps their language and rewrites the machinery under it.
