// 09_primes.cu: Prime number test and generation

fn is_prime(n: i32) -> bool {
    if n <= 1 {
        return false
    }
    if n <= 3 {
        return true
    }
    if (n % 2) == 0 || (n % 3) == 0 {
        return false
    }

    mut i = 5
    for (i * i) <= n {
        if (n % i) == 0 || (n % (i + 2)) == 0 {
            return false
        }
        i += 6
    }
    return true
}

fn print_primes_up_to(limit: i32) {
    println("Prime numbers up to limit:")
    mut num = 2
    for num <= limit {
        if is_prime(num) {
            println(num)
        }
        num += 1
    }
}

fn main() {
    print_primes_up_to(30)
}
