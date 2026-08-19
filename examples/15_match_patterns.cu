// 15_match_patterns.cu: Pattern matching with integers, booleans, and wildcard defaults

fn describe_code(code: i32) {
    match code {
        200 => {
            println("OK: Success")
        }
        404 => {
            println("Not Found: Resource does not exist")
        }
        500 => {
            println("Server Error: Internal failure")
        }
        _ => {
            println("Unknown status code")
        }
    }
}

fn boolean_status(ready: bool) {
    match ready {
        true => {
            println("System is ready and operational")
        }
        false => {
            println("System is offline or loading")
        }
    }
}

fn main() {
    println("--- Testing HTTP Code Matching ---")
    describe_code(200)
    describe_code(404)
    describe_code(500)
    describe_code(418)

    println("--- Testing Boolean Pattern Matching ---")
    boolean_status(true)
    boolean_status(false)
}
