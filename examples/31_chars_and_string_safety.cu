// 31_chars_and_string_safety.cu: Demonstrating first-class char, safe string operations, and conversions

fn test_char_basics() {
    println("=== Char Literals and Comparisons ===")
    let ch1: char = 'A'
    let ch2: char = 'B'
    
    println("ch1:")
    println(ch1)
    println("ch2:")
    println(ch2)
    
    let is_less = ch1 < ch2
    println("Is 'A' < 'B'?")
    println(is_less)
    
    let is_same = ch1 == 'A'
    println("Is ch1 == 'A'?")
    println(is_same)
}

fn test_char_conversions() {
    println("=== Char Conversions & String Building ===")
    let code: i64 = 67
    let from_int = char(code)
    println("char(67):")
    println(from_int)
    
    let back_to_int = i64(from_int)
    println("i64('C'):")
    println(back_to_int)
    
    let as_str = string(from_int)
    println("string('C') + \"upid\":")
    println(as_str + "upid")
}

fn test_string_indexing_and_slicing() {
    println("=== String Indexing & Safe Slicing ===")
    let text = "Systems Programming in Cupid"
    println("Original text:")
    println(text)
    println("Length:")
    println(len(text))
    
    let first_char = text[0]
    let tenth_char = text[8]
    println("text[0]:")
    println(first_char)
    println("text[8]:")
    println(tenth_char)
    
    let word1 = text[0:7]
    let word2 = text[8:19]
    let word3 = text[23:28]
    println("Slice 0..7:")
    println(word1)
    println("Slice 8..19:")
    println(word2)
    println("Slice 23..28:")
    println(word3)
}

fn main() {
    test_char_basics()
    test_char_conversions()
    test_string_indexing_and_slicing()
    println("=== All string & char tests completed successfully! ===")
}
