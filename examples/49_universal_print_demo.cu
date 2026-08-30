// 49_universal_print_demo.cu
// Demonstrates upgraded universal print and println functions
// capable of directly printing structs, tuples, pointers, arrays, and multiple arguments.

struct Vector3 {
    x: i64
    y: i64
    z: i64
}

struct UserProfile {
    id: i64
    name: string
    is_admin: bool
    score: f64
}

fn main() {
    println("==================================================")
    println("   CUPID UNIVERSAL PRINT & PRINTLN DEMO           ")
    println("==================================================")

    // 1. Primitive printing with multiple arguments
    println("Integer: ", 42, ", Float: ", 3.14159, ", Bool: ", true, ", Char: ", 'Z')

    // 2. Struct printing by value
    let v = Vector3{ x: 10, y: 20, z: 30 }
    println("Struct Vector3 by value:")
    println(v)

    let user = UserProfile{
        id: 1001
        name: "Alice Developer"
        is_admin: true
        score: 99.5
    }
    println("Struct UserProfile:")
    println(user)

    // 3. Struct printing by reference (&Struct)
    println("Pointer to UserProfile (&user):")
    println(&user)

    // 4. Tuple printing
    let tuple_sample = (123, "Cupid Tuple", false, 45.67)
    println("Tuple (mixed types):")
    println(tuple_sample)

    // 5. Array printing
    let numbers = [10, 20, 30, 40, 50]
    println("Array of integers:")
    println(numbers)

    let names = ["Alpha", "Beta", "Gamma"]
    println("Array of strings:")
    println(names)

    println("==================================================")
    println("   UNIVERSAL PRINT DEMO SUCCESSFUL!               ")
    println("==================================================")
}
