// Example 42: Tuple Structs - Declaration, instantiation, field access, and methods

struct Color(u8, u8, u8)
struct Point2D(i64, i64)

fn (c: &Color) red() -> u8 {
    return c.0
}

fn (c: &Color) green() -> u8 {
    return c.1
}

fn (c: &Color) blue() -> u8 {
    return c.2
}

fn (p: &Point2D) sum() -> i64 {
    return p.0 + p.1
}

fn main() {
    println("=== 1. Tuple Struct Instantiation & Field Access ===")
    let rgb = Color(255, 128, 64)
    println("RGB Channels:")
    println(rgb.0)
    println(rgb.1)
    println(rgb.2)

    println("=== 2. Tuple Struct Receiver Methods ===")
    println("Methods on &Color:")
    println(rgb.red())
    println(rgb.green())
    println(rgb.blue())

    println("=== 3. Point2D Tuple Struct ===")
    let pt = Point2D(40, 60)
    println("Point coords:")
    println(pt.0)
    println(pt.1)
    println("Point sum:")
    println(pt.sum())

    println("=== 4. Tuple Struct Destructuring ===")
    let Point2D(px, py) = pt
    println("Destructured Point2D:")
    println(px)
    println(py)
}
