// 18_let_immutability.cu: Testing strict immutability (let) vs mutability (mut)

fn mutate_counter(mut counter: i32) -> i32 {
    counter += 10
    return counter
}

fn main() {
    println("--- Testing let vs mut ---")
    let immutable_val = 42
    // immutable_val+=2
    println("Immutable value:")
    println(immutable_val)

    mut mutable_val = 100
    println("Initial mutable value:")
    println(mutable_val)

    mutable_val += 50
    println("Mutated value (100 + 50):")
    println(mutable_val)

    mutable_val = 999
    println("Reassigned mutable value:")
    println(mutable_val)

    let updated = mutate_counter(mutable_val)
    println("Function mutated copy:")
    println(updated)
}
