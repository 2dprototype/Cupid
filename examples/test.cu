// find the bug and give me solution
// also check: examples/test.cu and examples/test.asm to find the bug quickly

fn greet(name: &string) {
    println("test")
    let h = "Hello, " + *name // this line craseshes
    println(h) // this prints only "Hello, "
}

fn main() {
    let title = "Cupid Language Spec"

    println("=== Borrowing &string ===")
    greet(&title)

    println("=== END ===")
}
