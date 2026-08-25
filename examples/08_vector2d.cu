// 08_vector2d.cu: 2D Vector math using borrow references
// Demonstrates how immutable borrowing (&Vec2) allows variables to be used multiple times without move.

struct Vec2 {
    x: i64
    y: i64
}

fn vec2_add(v1: &Vec2, v2: &Vec2) -> Vec2 {
    return Vec2{
        x: v1.x + v2.x
        y: v1.y + v2.y
    }
}

fn vec2_dot(v1: &Vec2, v2: &Vec2) -> i64 {
    return (v1.x * v2.x) + (v1.y * v2.y)
}

fn vec2_length_squared(v: &Vec2) -> i64 {
    return (v.x * v.x) + (v.y * v.y)
}

fn main() {
    let a = Vec2{ x: 3, y: 4 }
    let b = Vec2{ x: 1, y: 2 }

    // Borrowing 'a' and 'b' with &a and &b prevents moving ownership
    let sum = vec2_add(&a, &b)
    println("Vector sum x:")
    println(sum.x)
    println("Vector sum y:")
    println(sum.y)

    let dot = vec2_dot(&a, &b)
    println("Dot product:")
    println(dot)

    let len_sq = vec2_length_squared(&a)
    println("Length squared of (3,4):")
    println(len_sq)
}
