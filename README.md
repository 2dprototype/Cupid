# Cupid

A strictly statically typed, compiled systems programming language targeting native x86-64 machine code via FASM (Flat Assembler), featuring Go-inspired concurrency, compile-time memory safety, and zero garbage collection.

---

## Key Highlights

- **Direct Native Code** — Generates optimized x86-64 assembly; no VM, no bytecode, no interpreter.
- **Strict Static Typing** — 18 primitive types (`i8`..`i64`, `u8`..`u64`, `int`, `uint`, `usize`, `isize`, `f32`, `f64`, `bool`, `string`, `char`, `void`) with zero implicit type coercion.
- **Zero Garbage Collection** — Deterministic compile-time ownership, borrowing, and lifetime inference.
- **Go-Style Concurrency** — Native lightweight routines (`go`), strongly typed channels (`channel<T>`), and multiplexing (`select`).
- **Receiver Methods** — Go-inspired explicit receiver syntax (`fn (r: &mut Type) method()`).
- **Pattern Matching & Error Handling** — Exhaustive `match` expressions, `Option<T>` / `Result<T, E>`, and the `?` error unwrapping operator.
- **Hardware & Low-Level Control** — Explicit `unsafe` blocks and direct inline assembly (`asm { ... }`).
- **Standard Library in Pure Cupid** — Core algorithms, collections, math, formatting, and OS bridges written primarily in Cupid.

---

## Compiler Architecture

```text
Source (.cu)
    │
    ▼
  Lexer ────────► Token Stream
    │
    ▼
  Parser ───────► AST (Abstract Syntax Tree)
    │
    ▼
Module & Symbol Resolver ──► Module Graph & Scopes
    │
    ▼
Type Checker ───► Static Typing & Monomorphization
    │
    ▼
Borrow Checker ─► Ownership & Memory Safety Verification
    │
    ▼
  HIR ──────────► Desugared High-Level IR
    │
    ▼
  MIR ──────────► Mid-Level Control Flow IR
    │
    ▼
Optimizer ──────► Constant Folding, DCE, Inlining
    │
    ▼
Native Backend ─► x86-64 FASM Assembly (.asm)
    │
    ▼
Native Executable (.exe)
```

---

## Quick Start

### Prerequisites
- **Go 1.24+** (to build the compiler)
- **FASM** (`fasm/` directory with `FASM.EXE` included in repo)

### Building the Compiler
```bash
# Build the cupid compiler
go build -o cupid.exe .
```

### Running Programs
```bash
# Compile and run immediately
cupid run examples/01_hello.cu

# Build a native release executable
cupid build examples/07_fibonacci.cu --release

# Preserve generated x86-64 assembly
cupid build examples/25_receiver_methods.cu --emit-asm
```

---

## Language Syntax & Examples

### 1. Variables & Immutability
```cu
// Immutable by default
let name: string = "Cupid"
let count = 42

// Explicitly mutable
mut score: i64 = 0
score += 10

// Compile-time constant
const MAX_LIMIT: i64 = 1000
```

### 2. Functions & Go-Style Receivers
```cu
struct Point {
    x: i64
    y: i64
}

// Value receiver
fn (p: Point) distance_sq() -> i64 {
    return p.x * p.x + p.y * p.y
}

// Mutable reference receiver
fn (p: &mut Point) translate(dx: i64, dy: i64) {
    p.x += dx
    p.y += dy
}

fn main() {
    mut pt = Point{ x: 3, y: 4 }
    pt.translate(2, 6)
    println(pt.distance_sq())
}
```

### 3. Control Flow & Loops
```cu
fn main() {
    let score = 85
    if score >= 90 {
        println("Grade: A")
    } else {
        println("Grade: B")
    }

    // Unified for loop
    mut i = 0
    for i < 5 {
        if i == 2 {
            i += 1
            continue
        }
        println(i)
        i += 1
    }
}
```

### 4. Pattern Matching
```cu
fn handle_status(code: i64) {
    match code {
        200 => {
            println("OK: Success")
        }
        404 => {
            println("Error: Not Found")
        }
        _ => {
            println("Other status code")
        }
    }
}
```

### 5. Concurrency (`go`, `channel`, `select`)
```cu
mut ch = channel<i64>()

fn worker(id: i64) {
    Sleep(20)
    ch.send(id * 100)
}

fn main() {
    go worker(5)
    
    select {
        case val = ch.recv():
            println("Received from worker:")
            println(val)
    }
}
```

### 6. Modules & Standard Library
```cu
import "math" as m
import { to_upper, trim_space } from "strings"

fn main() {
    let max_val = m.max(10, 50)
    let text = to_upper("hello cupid")
    println(max_val)
    println(text)
}
```

### 7. Inline Assembly & Unsafe
```cu
fn main() {
    unsafe {
        asm {
            mov rax, 100
            add rax, 23
            mov rcx, rax
            call _cupid_print_i64
            call _cupid_println
        }
    }
}
```

---

## Standard Library Modules (`stdlib/`)

| Module | Description |
| :--- | :--- |
| `core` | Fundamental traits (`Comparable<T>`, `Equatable<T>`, `Clone<T>`), `Option<T>`, `Result<T, E>` |
| `math` | Mathematical functions (`abs`, `min`, `max`, `clamp`, `PI`, `E`) |
| `strings` | String manipulation (`to_upper`, `to_lower`, `trim_space`, `contains`, `index_of`, `split`) |
| `collections` | Generic data structures (`Vector<T>`, `HashMap<K, V>`) |
| `os` | Operating system interaction and process control (`exit`) |
| `fmt` | Formatted output and printing |
| `sync` | Synchronization primitives (`Mutex`, `RwMutex`) |

---

## Project Structure

```text
cupid/
├── compiler/
│   ├── ast/           # AST node definitions
│   ├── backend/       # FASM x86-64 native code emitter
│   ├── borrow/        # Ownership and borrow checker
│   ├── diagnostics/   # Error reporting with source code spans
│   ├── driver/        # Compiler pipeline driver
│   ├── hir/           # High-level intermediate representation
│   ├── lexer/         # UTF-8 token stream lexer
│   ├── mir/           # Control-flow mid-level IR
│   ├── modules/       # Module dependency resolution
│   ├── optimizer/     # SSA/MIR optimizations
│   ├── parser/        # Handwritten recursive-descent parser
│   ├── resolver/      # Lexical scoping and symbol resolution
│   └── types/         # Strict type checking & monomorphization
├── stdlib/            # Standard library in pure Cupid
├── examples/          # 35+ verified runnable examples
├── fasm/              # Flat Assembler binaries and includes
├── main.go            # CLI entry point
└── plan.md            # Master language blueprint
```

---

## License

Cupid is licensed under the MIT License.
