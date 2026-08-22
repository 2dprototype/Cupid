// 24_pointers_references.cu: Testing safe references &T, &mut T, and dereferencing

fn increment(val_ref: &mut i64) {
    *val_ref += 1
}

fn add_amounts(a: &i64, b: &i64) -> i64 {
    return *a + *b
}

fn main() {
    println("--- Testing References and Dereferencing ---")
    mut count: i64 = 42
    println("Initial count:")
    println(count)

    let r: &i64 = &count
    println("Value through immutable reference *r:")
    println(*r)

    println("Calling increment(&mut count)...")
    increment(&mut count)
    println("Count after increment:")
    println(count)

    println("Calling increment(&mut count) again...")
    increment(&mut count)
    println("Count after second increment:")
    println(count)

    let x: i64 = 100
    let y: i64 = 200
    let total = add_amounts(&x, &y)
    println("add_amounts(&100, &200):")
    println(total)
}
