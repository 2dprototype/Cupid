# Cupid

Strictly statically typed, compiled, systems programming language.

## What is Cupid

Cupid is an experimental systems programming language and compiler implemented in Go. It targets native x86-64 code via the FASM assembler and aims for strict static typing, performance, and explicit control over low-level details. The repository contains the compiler frontend, middle- and back-ends, a small standard library, and example programs.

## Key features

- Statically typed with a multi-stage compilation pipeline (AST → HIR → MIR → assembly)
- Borrow checking (inspired by modern safe-systems languages)
- Optimizer with release/debug modes
- Native code generation to FASM (x86-64)
- Small standard library (core, fmt, math, os, strings)

## Repository layout

Top-level entries you will find:

```
main.go           # CLI frontend (build/run/test/fmt)
go.mod            # Go module file (go 1.24.4)
compiler/         # Compiler implementation (ast, parser, types, hir, mir, backend, ...)
stdlib/           # Standard library modules (core, fmt, math, os, strings)
examples/         # Example Cupid programs
```

How it fits together: the `main.go` CLI uses `compiler/driver` to run the full compilation pipeline. The driver resolves modules, runs name resolution and type checking, performs borrow checking, lowers to MIR, runs the optimizer, and finally calls the backend to emit assembly and invoke FASM to produce a native executable.

## Getting started (from source)

Requirements

- Go 1.24 or later (source built with go 1.24.4 in go.mod)
- FASM (flat assembler) placed in a `fasm/` directory next to the input file or where the compiler can find it (`FASM.EXE` expected by the driver)

Build the compiler

```bash
# From the repository root
go build -o cupid .
# or install to $GOBIN
go install
```

Compile a Cupid program

```bash
# compile
./cupid build examples/hello.cu --output hello.exe

# compile + run
./cupid run examples/hello.cu
```

Other CLI commands

```bash
./cupid test            # run go tests for the compiler
./cupid fmt <file.cu>   # (placeholder) format a Cupid source file
./cupid help            # show usage
```

Flags supported by the CLI (see `main.go`):

- `--emit-asm` — keep generated `.asm` file next to the source
- `--output, -o <path>` — specify compiled executable path
- `--release` — enable optimizer/optimizations for release
- `--debug` — include debug information

## Standard library

The `stdlib/` directory contains small modules that the compiler will resolve when building programs. Notable packages:

- `stdlib/core` — core language primitives
- `stdlib/fmt`  — formatting utilities
- `stdlib/math` — math helpers
- `stdlib/os`   — OS interaction helpers
- `stdlib/strings` — string utilities

## Examples

See `examples/` for sample Cupid programs. Use the CLI `build` or `run` commands to compile and run them.

## Implementation notes (for contributors)

- The compiler entrypoint is `compiler/driver/driver.go`. The driver orchestrates module resolution, name and type checking, borrow checking, lowering, optimization, assembly generation, and invoking the assembler.
- Native backend emits FASM assembly using `compiler/backend`. The driver expects an assembler binary `FASM.EXE` in `fasm/`.
- The project is implemented in Go and uses a staged IR (HIR and MIR) under `compiler/hir` and `compiler/mir`.

## Contributing

Contributions are welcome. A few tips:

- Run `go test ./...` frequently while changing compiler internals
- Keep changes small and focused when modifying language semantics
- Add example programs to `examples/` demonstrating new features or fixes

## Known limitations

- Backend currently targets FASM/x86-64 and expects `FASM.EXE` (Windows-style filename). Cross-platform assembly/invocation may need work.
- The `fmt` command in the CLI is a placeholder and currently only prints a message.

## License

If there is no LICENSE file in the repository, add one or clarify licensing before publishing.

---

If you'd like, I can:
- Open a pull request that adds this README to the repository
- Expand the Usage section with concrete example Cupid source files from `examples/`
- Add a contributing guide and a LICENSE file
