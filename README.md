# Cupid

A strictly statically typed, compiled systems programming language implemented in Go, targeting native x86-64 code via FASM (Flat Assembler).

## Overview

Cupid is an experimental systems programming language designed to combine strict static typing, modern language features (borrow checking), and direct native code generation. The compiler is written in Go and produces optimized x86-64 assembly that can be directly executed or integrated into larger systems.

### Key Features

- **Strict Static Typing** – Full type safety at compile time with comprehensive type inference
- **Multi-Stage Compilation** – AST → HIR (High-level IR) → MIR (Mid-level IR) → FASM Assembly
- **Borrow Checking** – Memory safety inspired by modern systems languages
- **Native Code Generation** – Compiles directly to FASM x86-64 assembly
- **Release & Debug Modes** – Built-in optimizer with configurable compilation strategies
- **Modular Standard Library** – Core utilities, formatting, math, OS interaction, and strings
- **Rich Examples** – 20+ example programs demonstrating language features

---

## Repository Layout

```
main.go                # CLI entry point (build, run, test, fmt commands)
go.mod                 # Go 1.24.4 module definition

compiler/              # Full compiler implementation
  ├─ driver/          # Orchestration & compilation pipeline
  ├─ lexer/           # Tokenization
  ├─ parser/          # Parsing to AST
  ├─ ast/             # Abstract syntax tree definitions
  ├─ resolver/        # Name & symbol resolution
  ├─ types/           # Type checking & inference
  ├─ borrow/          # Borrow checker (memory safety)
  ├─ hir/             # High-level intermediate representation
  ├─ mir/             # Mid-level intermediate representation
  ├─ backend/         # FASM x86-64 code generation
  ├─ optimizer/       # IR optimization passes
  ├─ modules/         # Module system & resolution
  └─ diagnostics/     # Error reporting & formatting

stdlib/                # Standard library (resolved at compile time)
  ├─ core             # Language primitives
  ├─ fmt              # Formatted printing
  ├─ math             # Mathematical functions
  ├─ os               # Operating system interface
  └─ strings          # String utilities

examples/              # 20+ example programs
  └─ 01_hello.cu, 02_arithmetic.cu, ..., 20_all_primitive_types.cu
```

### How It Works

1. **Module Resolution** – The driver resolves all input files and standard library dependencies into a complete module graph
2. **Parsing & Name Resolution** – Source code is tokenized and parsed into an AST, then symbols are resolved against scope
3. **Type Checking** – Full type checking with inference; all types must be statically verifiable
4. **Borrow Checking** – Memory ownership rules are validated to prevent use-after-free and double-free
5. **IR Lowering** – AST is lowered to high-level IR, then mid-level IR for optimization
6. **Optimization** – Optional passes run in release mode to improve performance
7. **Code Generation** – MIR is emitted as FASM x86-64 assembly
8. **Assembly & Linking** – FASM assembler produces a native executable

---

## Getting Started

### Prerequisites

- **Go 1.24 or later** (project built with Go 1.24.4)
- **FASM (Flat Assembler)** – Downloaded and placed in a `fasm/` directory next to the compiler

### Building the Compiler

```bash
# Clone the repository
git clone https://github.com/2dprototype/Cupid.git
cd Cupid

# Build the compiler
go build -o cupid .

# Or install to $GOPATH/bin
go install
```

### Compiling Cupid Programs

```bash
# Compile to an executable
./cupid build examples/01_hello.cu --output hello.exe

# Compile and run immediately
./cupid run examples/02_arithmetic.cu

# Compile in release mode (with optimizations)
./cupid build examples/07_fibonacci.cu --release

# Preserve generated assembly file
./cupid build examples/08_vector2d.cu --emit-asm

# Include debug information
./cupid build examples/12_game_entity.cu --debug
```

### CLI Commands

```bash
./cupid build <file.cu> [flags]   # Compile a Cupid program
./cupid run <file.cu> [flags]     # Compile and execute
./cupid test                       # Run Go unit tests for the compiler
./cupid fmt <file.cu>              # Format a Cupid source file (placeholder)
./cupid help                       # Show usage information
```

### Flags

- `--emit-asm` – Keep the generated `.asm` file (useful for inspection)
- `--output, -o <path>` – Specify the output executable path
- `--release` – Enable optimizer and release-mode optimizations
- `--debug` – Include debugging symbols in the executable

---

## Language Examples

### Hello World

```rust
// 01_hello.cu
fn main() {
    println("Hello, Cupid!")
    println(42)
    println(true)
}
```

### Functions & Arithmetic

```rust
// 02_arithmetic.cu
fn add(a: i64, b: i64) -> i64 {
    return a + b
}

fn main() {
    let x = 10
    let y = 20
    let result = add(x, y)
    println(result)  // Output: 30
}
```

### Control Flow

```rust
// 03_control_flow.cu
fn main() {
    let n = 5
    if n > 0 {
        println("positive")
    } else {
        println("non-positive")
    }
    
    for i = 0; i < n; i = i + 1 {
        println(i)
    }
}
```

### Structs & Methods

```rust
// 04_structs_methods.cu
struct Point {
    x: i64
    y: i64
}

fn (p: Point) distance() -> i64 {
    return p.x + p.y
}

fn main() {
    let pt = Point { x: 3, y: 4 }
    println(pt.distance())
}
```

See `examples/` for 16+ additional examples covering arrays, pattern matching, option types, generics, and more.

---

## Standard Library

The `stdlib/` directory contains modules automatically resolved by the compiler:

- **core** – Fundamental types and primitives
- **fmt** – `println()`, formatted output
- **math** – Arithmetic and mathematical functions
- **os** – Process exit, environment interaction
- **strings** – String manipulation utilities

Import and use with:

```rust
use strings

fn main() {
    let s = strings::to_upper("hello")
    println(s)
}
```

---

## Development & Contributing

### Running Tests

```bash
go test ./...
```

Tests are located throughout the compiler packages and cover parser, type checker, borrow checker, and backend functionality.

### Contributing Guidelines

1. **Small, focused changes** – Keep PRs focused on a single feature or fix
2. **Test frequently** – Run `go test ./...` after modifying compiler internals
3. **Add examples** – Demonstrate new features with `.cu` files in `examples/`
4. **Follow Go conventions** – Use standard Go idioms and formatting (`gofmt`)

---

## Known Limitations & Future Work

- **Platform** – Backend currently targets FASM/x86-64 only (Windows-style `FASM.EXE`); cross-platform support may require additional work
- **Formatter** – The `cupid fmt` command is a placeholder
- **Standard Library** – Core library is minimal; community contributions welcome
- **Inline Assembly** – Supported but limited to x86-64 FASM syntax
- **Error Messages** – Diagnostics are functional but could be more detailed

---

## Architecture Highlights

### Compilation Pipeline

The `compiler/driver/driver.go` orchestrates all stages:

```
Input File
    ↓
Module Resolution (resolver/*) + Parsing (parser/*)
    ↓
Name & Symbol Resolution (resolver/*)
    ↓
Type Checking (types/*)
    ↓
Borrow Checking (borrow/*)
    ↓
AST → HIR Lowering (hir/*)
    ↓
HIR → MIR Lowering (mir/*)
    ↓
Optimization (optimizer/*) [release mode]
    ↓
Code Generation (backend/*) → .asm
    ↓
FASM Assembly → Executable
```

### Key Modules

| Module | Purpose |
|--------|---------|
| `lexer/` | Tokenizes Cupid source into tokens |
| `parser/` | Builds AST from tokens |
| `types/` | Type inference and checking |
| `resolver/` | Resolves names to definitions |
| `borrow/` | Validates memory ownership rules |
| `hir/` | High-level IR (closer to source) |
| `mir/` | Mid-level IR (closer to machine) |
| `backend/` | Generates FASM x86-64 assembly |
| `optimizer/` | Applies optimization passes |

---

## License

Add a LICENSE file (currently unspecified). Recommended: MIT, Apache 2.0, or GPL 3.0 for open-source projects.

---

## Questions & Support

- **Want to contribute?** Read the contributing section and open a PR
- **Found a bug?** Open an issue with a minimal example
- **Have a feature request?** Start a discussion or open an issue labeled `enhancement`

For more details on the compiler architecture, see the code comments in `compiler/driver/driver.go` and individual module READMEs (if present).

---

**Made with ❤️ by 2dprototype**
