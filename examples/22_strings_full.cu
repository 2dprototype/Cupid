// 22_strings_full.cu: Testing complete string operations, concatenation, indexing, slicing, and stdlib/strings

import {
    length,
    is_empty,
    char_at,
    substring,
    starts_with,
    ends_with,
    index_of,
    last_index_of,
    contains,
    count,
    repeat,
    to_upper,
    to_lower,
    replace,
    trim_space
} from "strings"

fn main() {
    println("--- Testing String Concatenation & Equality ---")
    let first = "Hello, "
    let second = "World!"
    let combined = first + second
    println(combined)

    let is_eq = combined == "Hello, World!"
    println("Is combined equal to 'Hello, World!'?")
    println(is_eq)

    let is_neq = combined != "Goodbye!"
    println("Is combined not equal to 'Goodbye!'?")
    println(is_neq)

    println("--- Testing String Indexing & Slicing ---")
    let greeting = "Cupid Language"
    let len_str = len(greeting)
    println("Length of greeting:")
    println(len_str)

    let byte0 = greeting[0]
    println("greeting[0] char ('C'):")
    println(byte0)
    println("greeting[0] as integer code point (67):")
    println(i64(byte0))

    let sub1 = greeting[0:5]
    println("greeting[0:5]:")
    println(sub1)

    let sub2 = greeting[6:14]
    println("greeting[6:14]:")
    println(sub2)

    println("--- Testing stdlib/strings Functions ---")
    let has_cupid = contains(greeting, "Cupid")
    println("contains(greeting, 'Cupid'):")
    println(has_cupid)

    let sw = starts_with(greeting, "Cup")
    println("starts_with(greeting, 'Cup'):")
    println(sw)

    let ew = ends_with(greeting, "age")
    println("ends_with(greeting, 'age'):")
    println(ew)

    let idx = index_of(greeting, "Lang")
    println("index_of(greeting, 'Lang'):")
    println(idx)

    let upper = to_upper("cupid rules")
    println("to_upper('cupid rules'):")
    println(upper)

    let lower = to_lower("CUPID RULES")
    println("to_lower('CUPID RULES'):")
    println(lower)

    let rep = repeat("Abc-", 3)
    println("repeat('Abc-', 3):")
    println(rep)

    let rep_str = replace("banana", "a", "o")
    println("replace('banana', 'a', 'o'):")
    println(rep_str)

    let padded = "   Trimmed Text   "
    let trimmed = trim_space(padded)
    println("trim_space('   Trimmed Text   '):")
    println(trimmed)
}
