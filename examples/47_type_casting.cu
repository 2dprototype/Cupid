// 47_type_casting.cu
// Demonstrates explicit type casting with the 'as' keyword

fn main() {
    println("==================================================")
    println("   CUPID EXPLICIT TYPE CASTING ('as') DEMO        ")
    println("==================================================")

    // Integer to Float casting
    let int_num: i64 = 42
    let float_num: f64 = int_num as f64
    println("i64 to f64 casting:")
    println(float_num)

    // Float to Integer truncation
    let pi_approx: f64 = 3.999
    let truncated: i64 = pi_approx as i64
    println("f64 to i64 truncation (expected 3):")
    println(truncated)

    // Integer widening and narrowing
    let small: i32 = 1000
    let big: i64 = small as i64
    let back: i32 = big as i32
    println("i32 -> i64 -> i32 roundtrip:")
    println(big)
    println(back)

    // Casting in arithmetic expressions
    let val_a: i64 = 15
    let val_b: i64 = 4
    let ratio: f64 = (val_a as f64) / (val_b as f64)
    println("15 as f64 / 4 as f64 (expected 3.75):")
    println(ratio)

    println("==================================================")
    println("   EXPLICIT TYPE CASTING DEMO SUCCESSFUL!         ")
    println("==================================================")
}
