// test_pointers_strings.cu

// 1. Deref + concat
fn concat_greet(greeting: &string, name: &string) -> string {
    println("start")
    let c = *greeting + *name
    println(greeting + name)
    return c
}

// 2. Slicing a borrowed string
fn first_word(s: &string) -> string {
    mut i = 0
    let n = len(*s)
    for i < n {
        if (*s)[i] == ' ' { break }
        i += 1
    }
    return (*s)[0:i]
}

// 3. Mutating through a mutable reference
fn add_one(x: &mut i64) {
    *x += 1
}

fn main() {
    let hello = "Hello, "
    let world = "World!"
    let msg = concat_greet(&hello, &world)
    println(msg)   // "Hello, World!"

    let title = "Cupid Language Spec"
    let word = first_word(&title)
    println(word)  // "Cupid"

    mut n = 10
    add_one(&mut n)
    println(n)     // 11

    // len() on strings and slices
    println(len(title)) // 21
    println(len(word))  // 5
}