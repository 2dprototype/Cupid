// 21_floats_and_number_systems.cu: Testing complete float support and number systems in Cupid

fn main() {
    println("--- Testing Float Declarations & typeof ---")
    mut a: f32 = 20
    println("typeof(a):")
    println(typeof(a))
    println("Value of a:")
    println(a)

    mut b: f32 = 20.0
    println("typeof(b):")
    println(typeof(b))
    println("Value of b:")
    println(b)

    let c: f64 = 3.14159
    println("typeof(c):")
    println(typeof(c))
    println("Value of c:")
    println(c)

    println("--- Testing Float Arithmetic ---")
    let sum: f64 = 10.5 + 4.5
    println("10.5 + 4.5:")
    println(sum)

    let diff: f64 = 20.0 - 5.5
    println("20.0 - 5.5:")
    println(diff)

    let prod: f64 = 2.5 * 4.0
    println("2.5 * 4.0:")
    println(prod)

    let quot: f64 = 15.0 / 2.0
    println("15.0 / 2.0:")
    println(quot)

    let neg: f64 = -sum
    println("Negated sum:")
    println(neg)

    println("--- Testing Number Systems & Literals ---")
    let hex_val: i64 = 0xFF
    println("Hex 0xFF (255):")
    println(hex_val)

    let bin_val: i64 = 0b1010
    println("Binary 0b1010 (10):")
    println(bin_val)

    let oct_val: i64 = 0o77
    println("Octal 0o77 (63):")
    println(oct_val)

    let underscored: i64 = 1_000_000
    println("Underscored 1_000_000:")
    println(underscored)

    let sci_val: f64 = 1e3
    println("Scientific 1e3 (1000):")
    println(sci_val)
}
