// 23_arrays_slices.cu: Testing arrays, indexing, modifications, and slicing

fn main() {
    println("--- Testing Fixed Size Arrays ---")
    mut numbers: [5]i64 = [10, 20, 30, 40, 50]
    
    println("Length of numbers array:")
    println(len(numbers))

    println("numbers[0]:")
    println(numbers[0])

    println("numbers[2]:")
    println(numbers[2])

    println("Modifying numbers[2] from 30 to 999...")
    numbers[2] = 999
    println("New numbers[2]:")
    println(numbers[2])

    mut sum: i64 = 0
    mut i: i64 = 0
    for i < 5 {
        sum += numbers[i]
        i += 1
    }
    println("Sum of all numbers:")
    println(sum)

    println("--- Testing Array Slicing ---")
    let sub_slice = numbers[1:4]
    println("sub_slice[0] (should be 20):")
    println(sub_slice[0])
    println("sub_slice[1] (should be 999):")
    println(sub_slice[1])
    println("sub_slice[2] (should be 40):")
    println(sub_slice[2])
}
