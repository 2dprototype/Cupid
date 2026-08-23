// 29_select_multiplex.cu: Channel multiplexing with select

mut ch1 = channel<i64>()

fn sender() {
    Sleep(20)
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
