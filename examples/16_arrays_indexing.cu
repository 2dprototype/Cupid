// 16_arrays_indexing.cu: Array literal creation, element access, mutation, and len()

fn print_array_sum(arr: [4]i64) -> i64 {
    mut total = 0
    mut i = 0
    let n = len(arr)
    for i < n {
        total += arr[i]
        i += 1
    }
    return total
}

fn main() {
    println("--- Testing Fixed Size Arrays ---")
    mut scores: [4]i64 = [10, 25, 40, 50]

    println("First score:")
    println(scores[0])

    println("Third score:")
    println(scores[2])

    println("Updating second score from 25 to 99...")
    scores[1] = 99
    println("New second score:")
    println(scores[1])

    let sum = print_array_sum(scores)
    println("Total sum of scores:")
    println(sum)
}
