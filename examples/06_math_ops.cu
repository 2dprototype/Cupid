// 06_math_ops.cu: Arithmetic, bitwise operations, and comparison logic

fn is_even(n: i32) -> bool {
    return (n % 2) == 0
}

fn min(a: i32, b: i32) -> i32 {
    if a < b {
        return a
    }
    return b
}

fn max(a: i32, b: i32) -> i32 {
    if a > b {
        return a
    }
    return b
}

fn main() {
    let x = 16
    let y = 42

    let smaller = min(x, y)
    let larger = max(x, y)

    println(smaller)
    println(larger)
    println(is_even(x))
    println(is_even(y - 1))
}
