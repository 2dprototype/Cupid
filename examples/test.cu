// 04_structs_methods.cu
struct Point {
    x: i64
    y: i64
}

fn (p: Point) distance() -> i64 {
    return p.x + p.y
}

fn main() {
    let pt = Point { x: 3, y: 4 }
    println(pt.distance())
}