// 38_string_move_and_borrow.cu: 16-byte {ptr, len} string layout, zero-copy slicing, and borrowing

fn greet(name: &string) {
    println("Hello, " + *name)
}

fn first_word(s: &string) -> string {
    mut i = 0
    let str_len = len(*s)
    for i < str_len {
        if (*s)[i] == ' ' {
            break
        }
        i += 1
    }
    return (*s)[0:i] // zero-copy borrowed slice into the existing string buffer
}

fn main() {
    let title = "Cupid Language Spec"

    println("=== Borrowing &string ===")
    greet(&title)

    println("=== Zero-Copy Slicing ===")
    let word = first_word(&title)
    println(word)

    println("=== O(1) len() Reading ===")
    println(len(title))
    println(len(word))

    println("=== Move Semantics ===")
    let owner = title // MOVE: title is moved into owner
    println(owner)
}
