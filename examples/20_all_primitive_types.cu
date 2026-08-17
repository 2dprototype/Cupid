// 20_all_primitive_types.cu: Testing all available integer, float, bool, string, and composite types

struct Point {
    x i64
    y i64
}

fn test_primitives(a: i64, b: u64, flag: bool, msg: string) {
    println(msg)
    println(a)
    println(b)
    println(flag)
}

fn main() {
    println("--- Testing Cupid Type System ---")
    let small_i: i64 = -120
    let big_u: u64 = 18446744073
    let status: bool = true
    let greeting: string = "Cupid Type System Active!"

    // test_primitives(small_i, big_u, status, greeting)

    // println("--- Testing Struct Compound Type ---")
    // let pt: Point = Point{ x: 100, y: 200 }
    // println("Point X:")
    // println(pt.x)
    // println("Point Y:")
    // println(pt.y)

    // println("--- Testing Array Types ---")
    // let numbers: [3]i64 = [1, 2, 3]
    // println("Array element 0:")
    // println(numbers[0])
    // println("Array element 2:")
    // println(numbers[2])
}
