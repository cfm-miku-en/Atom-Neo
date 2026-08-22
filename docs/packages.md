# Making a package

Back to the [README](../README.md) or the [reference](reference.md).

A package is a folder with an `atom.json` in it. That is the whole format.

```
greet/
├── atom.json
└── greet.atom
```

---

## The manifest

```json
{
  "name": "greet",
  "version": "1.0.0",
  "description": "friendly greetings",
  "main": "greet.atom"
}
```

| field | |
|---|---|
| `name` | what people `import`. Required. |
| `version` | required |
| `description` | optional |
| `main` | the `.atom` file to run on import |
| `native` | instead of `main`, bind to a module built into the interpreter |

A package needs either `main` or `native`, not both.

Names ignore case, and `-`, `_` and `.` all collapse together. `My-Thing`, `my_thing` and
`my.thing` are the same package, so two of them cannot differ only by punctuation.

---

## Walkthrough

There is a working copy of this in [`packages/greet`](../packages/greet).

**1. Make the folder**

```
packages/greet/atom.json
packages/greet/greet.atom
```

**2. Write the manifest.** The json above.

**3. Write the code.** Anything you define at the top level becomes available to whoever
imports it.

```
func greeting(name) {
	return ["hello " + name]
}

func shout(name) {
	return [upper(greeting(name))]
}
```

**4. Install it**

```bash
atom install packages/greet
```

That copies it into `atom_modules/greet`.

**5. Use it**

```
import greet

print([greeting("world")])
print([shout("atom")])
```

```
hello world
HELLO ATOM
```

---

## What import actually does

`import greet` finds `atom_modules/greet`, reads its `atom.json`, and runs the file named by
`main`. Running it defines its functions, which is what makes them callable afterwards.

Two things follow from that:

- **Top level code runs on import.** A `print` at the top level of a package fires the
  moment someone imports it, so keep the top level to definitions.
- **There is one namespace.** A package function named `len` would shadow the builtin for
  the whole program. Prefixing names, like `greet_shout`, avoids surprises.

Importing the same package twice only runs it once.

---

## Shipping it

A package can be shared as the folder, or zipped:

```bash
# from inside the package folder
zip -r ../greet.zip .
```

Either installs the same way:

```bash
atom install greet.zip
atom install packages/greet
```

Keep `atom.json` at the top of the zip, or in a single folder inside it. Both layouts work.

---

## Native packages

Some things cannot be written in Atom. `web` needs Go's `net/http`, so it ships compiled
into the interpreter and its package is only a binding:

```json
{
  "name": "web",
  "version": "0.1.0",
  "description": "HTTP server backed by the compiled-in web module",
  "native": "web"
}
```

There is no `main`. On import, Atom looks for a native module registered under that name and
activates it, which is what makes `web.listen` callable. If the interpreter was built
without it, the import fails with a clear message rather than a missing function later.

Adding one means writing Go in `src/stdlib/` and registering it in `src/stdlib/stdlib.go`.
See [`src/stdlib/web`](../src/stdlib/web) for the shape. Native modules cannot be
downloaded and loaded at runtime: Go has no portable way to load compiled code into a
running program, so they are built in.

---

## Layout

Nothing stops a package from having more than one file, but only `main` is run. Anything
else has to be pulled in by it, and there is no mechanism for that yet, so a package is
effectively one `.atom` file plus whatever data it reads.
