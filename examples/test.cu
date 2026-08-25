fn print_num(num: &i64) {
    println(*num) // i think the problem crasesh in this line for some reason
}

// fn print_num(num: &i64) -> void {
    // println(*num) // but somehow this works when i put return type as void
// }

// fn print_num(num: &i64) {
    // println(num) // even this doesnt crash when we dont use *num
// }

fn main() {
    println("Printing num...")
    print_num(&100)
    println("End")
}