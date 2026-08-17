// 17_option_result.cu: Safe error handling using Option<T> and Result<T, E>

import { Option, Some, None, Result, Ok, Err } from "core"

fn safe_divide(numerator: i64, denominator: i64) -> Option<i64> {
    if denominator == 0 {
        return None<i64>()
    }
    return Some<i64>(numerator / denominator)
}

fn validate_user_age(age: i64) -> Result<i64, string> {
    if age < 0 {
        return Err<i64, string>("Age cannot be negative")
    }
    if age > 150 {
        return Err<i64, string>("Age exceeds maximum human lifespan")
    }
    return Ok<i64, string>(age)
}

fn main() {
    println("--- Testing Option ---")
    let valid_div = safe_divide(100, 4)
    if valid_div.is_some {
        println("100 / 4 result:")
        println(valid_div.value)
    } 
    else {
        println("Division by zero occurred")
    }

    let zero_div = safe_divide(100, 0)
    if zero_div.is_some {
        println(zero_div.value)
    }
    else {
        println("Division by zero safely handled!")
    }

    println("--- Testing Result ---")
    let good_age = validate_user_age(28)
    if good_age.is_ok {
        println("User age validated successfully:")
        println(good_age.value)
    }
    else {
        println(good_age.error)
    }

    let bad_age = validate_user_age(-5)
    if bad_age.is_ok {
        println(bad_age.value)
    } 
    else {
        println("Error occurred during validation:")
        println(bad_age.error)
    }
}
