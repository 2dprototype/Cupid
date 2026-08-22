// 02_arithmetic.cu: Functions, arithmetic operations, and expressions

fn add(a: i32, b: i32) -> i32 {
    return a + b
}

fn multiply(a: i32, b: i32) -> i32 {
    return a * b
}

fn compute(x: i32, y: i32) -> i32 {
    let sum = add(x, y)
    let prod = multiply(x, y)
    return prod - sum
}

fn main() {
    let a: i32 = 10
    let b: i32 = 25
    let result = compute(a, b)
    println(result)
}
