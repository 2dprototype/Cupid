// 19_typed_arrays.cu: Explicit array type annotations and validation

fn sum_fixed_array(arr: [4]i64) -> i64 {
    mut sum = 0
    mut i = 0
    let n = len(arr)
    for i < n {
        sum += arr[i]
        i += 1
    }
    return sum
}

fn main() {
    println("--- Testing Explicit Typed Array Declarations ---")
    mut scores: [4]i64 = [10, 25, 40, 50]

    println("First element:")
    println(scores[0])

    println("Modifying index 1 to 75...")
    scores[1] = 75
    println("Modified element:")
    println(scores[1])

    let total = sum_fixed_array(scores)
    println("Total array sum (10 + 75 + 40 + 50):")
    println(total)
}
