// 35_all_primitive_types_complete.cu: Comprehensive test of all primitive types in Cupid
// Tests: i8, i16, i32, i64, u8, u16, u32, u64, int, uint, usize, isize, f32, f64, bool, string, char, void

// 1. Explicit void return function
fn test_void_explicit() -> void {
    println("Void function executed successfully.")
    return
}

// 2. Implicit void return function
fn test_void_implicit() {
    println("Implicit void return function executed.")
}

// 3. Testing signed integers
fn test_signed_integers() {
    println("=== 1. Signed Integers ===")
    let v_i8: i8 = -128
    let v_i16: i16 = -32768
    let v_i32: i32 = -2147483648
    let v_i64: i64 = -9223372036854775807
    let v_isize: isize = -1000
    let v_int: int = -50000

    println("i8:")
    println(v_i8)
    println("i16:")
    println(v_i16)
    println("i32:")
    println(v_i32)
    println("i64:")
    println(v_i64)
    println("isize:")
    println(v_isize)
    println("int:")
    println(v_int)
}

// 4. Testing unsigned integers
fn test_unsigned_integers() {
    println("=== 2. Unsigned Integers ===")
    let v_u8: u8 = 255
    let v_u16: u16 = 65535
    let v_u32: u32 = 4294967295
    let v_u64: u64 = 18446744073709551615
    let v_usize: usize = 1000000
    let v_uint: uint = 5000000

    println("u8:")
    println(v_u8)
    println("u16:")
    println(v_u16)
    println("u32:")
    println(v_u32)
    println("u64:")
    println(v_u64)
    println("usize:")
    println(v_usize)
    println("uint:")
    println(v_uint)
}

// 5. Testing floating-point numbers
fn test_floats() {
    println("=== 3. Floating-Point Types ===")
    let v_f32: f32 = 3.14
    let v_f64: f64 = 2.718281828459

    println("f32:")
    println(v_f32)
    println("f64:")
    println(v_f64)

    let float_sum: f64 = v_f64 + 1.0
    println("f64 + 1.0:")
    println(float_sum)
}

// 6. Testing boolean type
fn test_boolean() {
    println("=== 4. Boolean Type ===")
    let t: bool = true
    let f: bool = false

    println("true:")
    println(t)
    println("false:")
    println(f)
    println("true && false:")
    println(t && f)
    println("true || false:")
    println(t || f)
    println("!false:")
    println(!f)
}

// 7. Testing string and char
fn test_string_and_char() {
    println("=== 5. String and Char Types ===")
    let letter: char = 'C'
    let message: string = "Cupid Systems Language"

    println("char:")
    println(letter)
    println("string:")
    println(message)
    println("len(string):")
    println(len(message))
}

// 8. Testing type casts between primitives
fn test_casts() {
    println("=== 6. Explicit Primitive Type Casting ===")
    let a: i8 = 65
    let c: char = char(a)
    let as_i64: i64 = i64(a)
    let as_u32: u32 = u32(a)
    let as_f64: f64 = f64(a)

    println("i8(65) cast to char:")
    println(c)
    println("i8(65) cast to i64:")
    println(as_i64)
    println("i8(65) cast to u32:")
    println(as_u32)
    println("i8(65) cast to f64:")
    println(as_f64)
}

fn main() {
    println("#############################################")
    println("# Testing All 18 Cupid Primitive Types      #")
    println("#############################################")

    test_signed_integers()
    test_unsigned_integers()
    test_floats()
    test_boolean()
    test_string_and_char()
    test_casts()
    test_void_explicit()
    test_void_implicit()

    println("#############################################")
    println("# All Primitive Types Verified Successfully!#")
    println("#############################################")
}
