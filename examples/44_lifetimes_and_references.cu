// 44_lifetimes_and_references.cu
// Demonstrates lifetime annotations on functions, structs, and references,
// along with nested comments and multiline strings.

/* Outer comment block
   Resuming outer comment block
*/
/* Inner nested comment block */

struct Cursor<'a> {
    text: &'a string
    pos: i64
}

fn inspect_cursor<'a>(c: &Cursor<'a>) {
    println("Inspecting cursor:")
    println(c.text)
    println(c.pos)
}

fn longer<'a>(x: &'a string, y: &'a string) -> &'a string {
    if len(x) >= len(y) {
        return x
    }
    return y
}

fn main() {
    println("==================================================")
    println("   CUPID LIFETIMES & REFERENCE SYNTAX DEMO        ")
    println("==================================================")

    let msg1: string = "Hello"
    let msg2: string = "Greetings, Cupid World!"

    let chosen: &string = longer<'a>(&msg1, &msg2)
    println("Longer message:")
    println(chosen)

    let cur = Cursor<'a>{
        text: &msg2
        pos: 10
    }
    inspect_cursor<'a>(&cur)

    let multi_str = "Cupid line 1
Cupid line 2
Cupid line 3"
    println("Multi-line string output:")
    println(multi_str)

    println("==================================================")
    println("   LIFETIMES & REFERENCE DEMO SUCCESSFUL!         ")
    println("==================================================")
}
