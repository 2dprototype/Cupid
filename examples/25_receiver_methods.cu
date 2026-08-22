// 25_receiver_methods.cu: Testing Go-style struct receiver methods

struct Point {
    x: i64
    y: i64
}

// Value receiver
fn (p: Point) distance_sq() -> i64 {
    return p.x * p.x + p.y * p.y
}

fn (p: Point) sum_coords() -> i64 {
    return p.x + p.y
}

// Mutable reference receiver
fn (p: &mut Point) translate(dx: i64, dy: i64) {
    p.x += dx
    p.y += dy
}

fn (p: &mut Point) scale(factor: i64) {
    p.x *= factor
    p.y *= factor
}

struct Counter {
    val: i64
}

fn (c: &mut Counter) increment() {
    c.val += 1
}

fn (c: Counter) get() -> i64 {
    return c.val
}

fn main() {
    println("--- Testing Point Receiver Methods ---")
    mut pt = Point{ x: 3, y: 4 }

    println("Point sum_coords():")
    println(pt.sum_coords())

    println("Point distance_sq() (should be 25):")
    println(pt.distance_sq())

    println("Translating Point by (2, 6)...")
    pt.translate(2, 6)
    println("New x (5):")
    println(pt.x)
    println("New y (10):")
    println(pt.y)

    println("Scaling Point by 2...")
    pt.scale(2)
    println("Scaled x (10):")
    println(pt.x)
    println("Scaled y (20):")
    println(pt.y)

    println("--- Testing Counter Receiver Methods ---")
    mut c = Counter{ val: 100 }
    c.increment()
    c.increment()
    c.increment()
    println("Counter value after 3 increments (should be 103):")
    println(c.get())
}
