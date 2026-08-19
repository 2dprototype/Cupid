// 02_arithmetic.cu: Functions, arithmetic operations, and expressions

fn add(a: i64, b: i64) -> i64 {
    return a + b
}

fn multiply(a: i64, b: i64) -> i64 {
    return a * b
}

fn compute(x: i64, y: i64) -> i64 {
    let sum = add(x, y)
    let prod = multiply(x, y)
    return prod - sum
}

fn main() {
    let a: i64 = 10
    let b: i64 = 25
    let result = compute(a, b)
    println(result)
}
