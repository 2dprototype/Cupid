// 46_keywords_and_syntax_enhancements.cu
// Demonstrates new keywords and tokens: select with case/default, true/false boolean literals, in, and Rust-style self receiver methods

struct Item {
    id: i64
    name: string
    active: bool
}

// Receiver method declaration using Rust-like 'self' receiver
fn (self: &Item) display() {
    println("Displaying Item via self receiver method:")
    println(self.id)
    println(self.name)
    println(self.active)
}

// Another method calling self.display()
fn (self: &Item) print_info() {
    println("Printing Item summary:")
    self.display()
}

mut comm_ch = channel<string>()

fn background_task() {
    Sleep(20)
    comm_ch.send("Task payload ready")
}

fn main() {
    println("==================================================")
    println("   CUPID KEYWORDS & SYNTAX ENHANCEMENTS DEMO     ")
    println("==================================================")

    // Direct true/false keywords
    let flag_true: bool = true
    let flag_false: bool = false
    println("Booleans true/false directly:")
    println(flag_true)
    println(flag_false)

    // Struct with boolean literals
    let it = Item{
        id: 101
        name: "Cupid Systems Unit"
        active: true
    }
    
    // Call receiver methods using standard dot syntax
    it.display()
    it.print_info()

    // Concurrency with select (case, default)
    go background_task()

    select {
        case msg = comm_ch.recv():
            println("Received via select case:")
            println(msg)
    }

    println("==================================================")
    println("   KEYWORDS & SYNTAX DEMO SUCCESSFUL!             ")
    println("==================================================")
}
