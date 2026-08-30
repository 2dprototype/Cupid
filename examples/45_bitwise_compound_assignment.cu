// 45_bitwise_compound_assignment.cu
// Demonstrates compound bitwise assignment operators: &=, |=, ^=, <<=, >>=

fn main() {
    println("==================================================")
    println("   CUPID BITWISE COMPOUND ASSIGNMENTS DEMO        ")
    println("==================================================")

    mut val: i64 = 0b11110000

    println("Initial value:")
    println(val)

    // &= bitwise AND assign
    val &= 0b10101111
    println("After &= 0b10101111 (expected 160):")
    println(val)

    // |= bitwise OR assign
    val |= 0b00000101
    println("After |= 0b00000101 (expected 165):")
    println(val)

    // ^= bitwise XOR assign
    val ^= 0b00001111
    println("After ^= 0b00001111 (expected 170):")
    println(val)

    // <<= shift left assign
    mut shift_val: i64 = 1
    shift_val <<= 4
    println("After 1 <<= 4 (expected 16):")
    println(shift_val)

    // >>= shift right assign
    shift_val >>= 2
    println("After 16 >>= 2 (expected 4):")
    println(shift_val)

    println("==================================================")
    println("   BITWISE COMPOUND ASSIGNMENTS SUCCESSFUL!       ")
    println("==================================================")
}
