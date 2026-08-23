// 37_generic_functions_and_inference.cu: Testing generic functions with automatic type inference
// Functions and structs with generic type parameters <T> monomorphized at compile time

fn identity<T>(x: T) -> T {
    return x
}

fn min_val<T>(a: T, b: T) -> T {
    if a < b {
        return a
    }
    return b
}

fn max_val<T>(a: T, b: T) -> T {
    if a > b {
        return a
    }
    return b
}

struct Pair<T> {
    first: T
    second: T
}

fn (p: &Pair<T>) sum() -> T {
    return p.first + p.second
}

fn main() {
    println("=== Testing Generic Functions with Explicit Types ===")
    let a1 = min_val<i64>(100, 250)
    let a2 = max_val<i64>(100, 250)
    println(a1)
    println(a2)

    println("=== Testing Generic Functions with Automatic Type Inference ===")
    let b1 = min_val(50, 25)
    let b2 = max_val(50, 25)
    let b3 = identity(999)
    let b4 = identity("Cupid Generics Monomorphized!")
    println(b1)
    println(b2)
    println(b3)
    println(b4)

    println("=== Testing Generic Struct Pair<i64> ===")
    let p = Pair<i64>{ first: 30, second: 70 }
    println("Pair sum (30 + 70 = 100):")
    println(p.sum())
}
