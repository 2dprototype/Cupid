import "time"

mut ch1 = channel<i64>()

fn sender() {
    time.sleep_ms(20)
    ch1.send(999)
}

fn main() {
    println("Testing select statement with channel:")
    go sender()
    
    select {
        case msg = ch1.recv():
            println("Received message in select:")
            println(msg)
    }
}
