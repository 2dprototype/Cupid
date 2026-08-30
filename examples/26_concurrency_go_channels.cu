import "time"

mut ch = channel<i64>()

fn worker(id: i64) {
    println("Worker started concurrently!")
    time.sleep_ms(50)
    ch.send(42)
}

fn main() {
    println("Main starting thread...")
    go worker(1)
    
    let result = ch.recv()
    println("Main received from channel:")
    println(result)
}
