// 27_result_option_question.cu: Error handling with Result, Option, and ? operator

struct MyOption {
    is_some bool
    value i64
}

fn get_positive(x: i64) -> MyOption {
    if x > 0 {
        return MyOption{
            is_some: true
            value: x * 2
        }
    }
    return MyOption{
        is_some: false
        value: 0
    }
}

fn calculate(x: i64) -> i64 {
    let unwrapped = get_positive(x)?
    return unwrapped + 10
}

fn main() {
    println("Testing ? unwrapping on Option/Result:")
    let res = calculate(25)
    println(res)
}
