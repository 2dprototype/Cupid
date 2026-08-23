// 28_generics_monomorphization.cu: Generic functions and monomorphization

fn min_val<T>(a: T, b: T) -> T {
    if a < b {
        return a
    }
    return b
}

fn max_val<T>(a: T, b: T) -> T {
    if a > b {
        return a
    }
    return b
}

fn main() {
    println("Testing generic monomorphized functions:")
    let m1 = min_val<i64>(100, 250)
    let m2 = max_val<i64>(100, 250)
    println(m1)
    println(m2)
}
