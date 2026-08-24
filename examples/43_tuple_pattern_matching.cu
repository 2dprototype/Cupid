// Example 43: Pattern Matching on Tuples and Tuple Structs

struct Pair(i64, i64)

fn describe_point(p: (i64, i64)) {
    match p {
        (0, 0) => {
            println("Origin (0, 0)")
        }
        (0, y) => {
            println("On Y axis")
        }
        (x, 0) => {
            println("On X axis")
        }
        _ => {
            println("General point")
        }
    }
}

fn main() {
    println("=== Pattern Matching on Tuples ===")
    describe_point((0, 0))
    describe_point((0, 15))
    describe_point((25, 0))
    describe_point((10, 20))
}
