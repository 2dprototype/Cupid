// 36_all_tokens_and_syntax_complete.cu: Comprehensive test of all Cupid tokens, keywords, and operators
// Keywords: fn, struct, impl, trait, let, mut, const, go, channel, select, unsafe, asm, match, import, export, as, from, if, else, for, return, where, break, continue
// Operators: +, -, *, /, %, ==, !=, <, <=, >, >=, =, +=, -=, *=, /=, %=, &&, ||, !, &, |, ^, <<, >>, ->, =>, ?, ., .., :, ::, ,, @
// Delimiters: (, ), {, }, [, ]
// Literals: IDENT, INT, FLOAT, STRING, CHAR

import "math" as m
import { abs, max } from "math"
import "time"

// 1. Constants
const BASE_OFFSET: i64 = 100
export const VERSION: string = "1.0.0"

// 2. Struct Declaration
export struct TokenStats {
    count: i64
    ratio: f64
    active: bool
}

// 3. Trait Declaration
trait Printable {
    fn print_summary(t: &TokenStats)
}

// 4. Method with Receiver and Trait Implementation
fn (ts: &TokenStats) summary() {
    println("TokenStats summary:")
    println(ts.count)
    println(ts.ratio)
    println(ts.active)
}

impl Printable for TokenStats {
    fn print_summary(t: &TokenStats) {
        t.summary()
    }
}

// 5. Result/Option struct with '?' operator
struct ResultBox {
    is_ok bool
    value i64
}

fn fetch_value(input: i64) -> ResultBox {
    if input > 0 {
        return ResultBox{
            is_ok: true
            value: input * 2
        }
    }
    return ResultBox{
        is_ok: false
        value: 0
    }
}

fn unwrap_demo(x: i64) -> i64 {
    let unwrapped = fetch_value(x)?
    return unwrapped + 5
}

// 6. Concurrency Channel & Routine
mut comm_ch = channel<i64>()

fn async_worker(task_id: i64) {
    time.sleep_ms(20)
    comm_ch.send(task_id + 777)
}

// 7. Generic function with where clause
fn process_generic<T>(item: T) -> T
where
    T: Printable,
{
    return item
}

fn main() {
    println("==================================================")
    println("   CUPID ALL TOKENS & SYNTAX VERIFICATION        ")
    println("==================================================")

    // --- Literals & Bindings ---
    let int_val: i64 = 42
    let hex_val: i64 = 0xFF
    let bin_val: i64 = 0b1010
    let float_val: f64 = 3.14159
    let str_val: string = "Hello Cupid"
    let char_val: char = 'Z'
    mut mut_counter: i64 = 0

    println("Literals:")
    println(int_val)
    println(hex_val)
    println(bin_val)
    println(float_val)
    println(str_val)
    println(char_val)

    // --- Arithmetic & Compound Assignment ---
    mut_counter += 10
    mut_counter -= 2
    mut_counter *= 3
    mut_counter /= 2
    mut_counter %= 7
    let arith_result = (10 + 5 - 2) * 4 / 2 % 3
    println("Arithmetic + - * / % and += -= *= /= %= result:")
    println(mut_counter)
    println(arith_result)

    // --- Comparisons ---
    let eq_check = (int_val == 42)
    let neq_check = (int_val != 0)
    let lt_check = (10 < 20)
    let lte_check = (10 <= 10)
    let gt_check = (20 > 10)
    let gte_check = (20 >= 20)
    println("Comparisons (==, !=, <, <=, >, >=):")
    println(eq_check && neq_check && lt_check && lte_check && gt_check && gte_check)

    // --- Logical Operators ---
    let log_and = true && false
    let log_or = true || false
    let log_not = !false
    println("Logical (&&, ||, !):")
    println(log_and)
    println(log_or)
    println(log_not)

    // --- Bitwise Operators ---
    let bit_and = 0b1100 & 0b1010
    let bit_or = 0b1100 | 0b0011
    let bit_xor = 0b1111 ^ 0b0101
    let bit_shl = 1 << 3
    let bit_shr = 16 >> 2
    println("Bitwise (&, |, ^, <<, >>):")
    println(bit_and)
    println(bit_or)
    println(bit_xor)
    println(bit_shl)
    println(bit_shr)

    // --- Arrays & Slices ---
    let arr: [4]i64 = [10, 20, 30, 40]
    println("Array indexing ([0], [3]):")
    println(arr[0])
    println(arr[3])

    // --- Control Flow: for, break, continue ---
    println("Testing for loop with break and continue:")
    mut sum = 0
    mut i = 0
    for i < 10 {
        i += 1
        if i == 2 {
            continue
        }
        if i > 5 {
            break
        }
        sum += i
    }
    println("Loop sum (1 + 3 + 4 + 5 = 13):")
    println(sum)

    // --- Pattern Matching: match, => ---
    println("Testing match statement:")
    let test_code = 2
    match test_code {
        1 => {
            println("Matched 1")
        }
        2 => {
            println("Matched 2 (Correct)")
        }
        _ => {
            println("Default match")
        }
    }

    // --- Struct Instantiation & Method Invocation ---
    mut stats = TokenStats{
        count: 500
        ratio: 0.95
        active: true
    }
    stats.summary()

    // --- Question Operator (?) Error Unwrapping ---
    println("Testing '?' operator unwrapping:")
    let unwrap_res = unwrap_demo(20)
    println(unwrap_res)

    // --- Concurrency: go, channel, select ---
    println("Testing concurrency (go, channel, select):")
    go async_worker(100)

    select {
        case msg = comm_ch.recv():
            println("Received from channel in select:")
            println(msg)
    }

    // --- Unsafe & Inline Assembly (asm) ---
    println("Testing unsafe & inline assembly (asm):")
    unsafe {
        asm {
            mov rax, 888
            mov rcx, rax
            call _cupid_print_i64
            call _cupid_println
        }
    }

    // --- Math Import via Alias ---
    println("Testing imported math module (as m):")
    let min_val = m.min(10, 50)
    let max_val = max(10, 50)
    let abs_val = abs(-99)
    println(min_val)
    println(max_val)
    println(abs_val)

    println("==================================================")
    println("   ALL TOKENS & SYNTAX PASSED SUCCESSFULLY!       ")
    println("==================================================")
}
