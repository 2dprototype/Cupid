// 07_fibonacci.cu: Recursive and iterative Fibonacci sequence computation

fn fib_recursive(n: i64) -> i64 {
    if n <= 0 {
        return 0
    }
    if n == 1 {
        return 1
    }
    return fib_recursive(n - 1) + fib_recursive(n - 2)
}

fn print_fib_sequence(count: i64) {
    println("Fibonacci numbers:")
    mut a = 0
    mut b = 1
    mut i = 0
    for i < count {
        println(a)
        let next_val = a + b
        a = b
        b = next_val
        i += 1
    }
}

fn main() {
    print_fib_sequence(10)
    
    println("10th Fibonacci recursively:")
    let fib10 = fib_recursive(10)
    println(fib10)
}
