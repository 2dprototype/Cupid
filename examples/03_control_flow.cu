// 03_control_flow.cu: Conditional branching and loops

fn check_value(n: i64) {
    if n > 50 {
        println("Greater than 50")
    } else if n == 50 {
        println("Exactly 50")
    } else {
        println("Less than 50")
    }
}

fn factorial(n: i64) -> i64 {
    if n <= 1 {
        return 1
    }
    return n * factorial(n - 1)
}

fn main() {
    check_value(75)
    check_value(50)
    check_value(25)

    let fact5 = factorial(5)
    println(fact5)
}
