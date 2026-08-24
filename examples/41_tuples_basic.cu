// Example 41: Tuples - Basic instantiation, indexing, function arguments, returns, and destructuring

fn swap(pair: (i64, i64)) -> (i64, i64) {
    return (pair.1, pair.0)
}

fn create_person() -> (string, i64, bool) {
    return ("Alice", 28, true)
}

fn main() {
    println("=== 1. Basic Tuple Instantiation & Indexing ===")
    let t: (i64, string, bool) = (100, "Cupid", true)
    println(t.0)
    println(t.1)
    println(t.2)

    println("=== 2. Tuple Functions & Return Values ===")
    let original = (10, 20)
    let swapped = swap(original)
    println("Original pair:")
    println(original.0)
    println(original.1)
    println("Swapped pair:")
    println(swapped.0)
    println(swapped.1)

    println("=== 3. Tuple Destructuring in Let Bindings ===")
    let (name, age, is_active) = create_person()
    println("Name:")
    println(name)
    println("Age:")
    println(age)
    println("Is Active:")
    println(is_active)

    println("=== 4. Direct Destructuring ===")
    let (x, y) = (500, 600)
    println(x)
    println(y)
}
