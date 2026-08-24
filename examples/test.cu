fn greet(name: &string) {
    // println("ok?")
    println("Hello, " + *name)
}

fn main() {
    let title = "Cupid Language Spec"
    println("=== Borrowing &string ===")
    greet(&title)
    println("end")
}
