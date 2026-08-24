// 39_slice_len_and_zero_copy.cu: Fixed arrays [N]T (Copy) vs Slices []T (16-byte view, O(1) len)

fn sum_fixed(arr: [5]i64) -> i64 {
    mut total: i64 = 0
    mut i = 0
    for i < 5 {
        total += arr[i]
        i += 1
    }
    return total
}

fn sum_slice(arr: []i64) -> i64 {
    mut total: i64 = 0
    mut i = 0
    let slice_len = len(arr) // O(1) reading of slice header length
    for i < slice_len {
        total += arr[i]
        i += 1
    }
    return total
}

fn main() {
    let nums: [5]i64 = [10, 20, 30, 40, 50]

    println("=== Sum Fixed Array [5]i64 ===")
    println(sum_fixed(nums))

    println("=== Sum Sub-Slice nums[1:4] (20 + 30 + 40 = 90) ===")
    let sub = nums[1:4]
    println("Slice length:")
    println(len(sub))
    println("Slice sum:")
    println(sum_slice(sub))

    println("=== Full Slice nums[0:5] ===")
    let full = nums[0:5]
    println(sum_slice(full))
}
