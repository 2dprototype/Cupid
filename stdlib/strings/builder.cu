// Cupid Standard Library: strings/builder
// Ordinary struct for efficient string accumulation

export struct StringBuilder {
    data: []u8
    length: i64
    capacity: i64
}

export fn new_string_builder() -> StringBuilder {
    return StringBuilder{
        length: 0
        capacity: 0
    }
}

fn (sb: &StringBuilder) len() -> i64 {
    return sb.length
}

fn (sb: &StringBuilder) is_empty() -> bool {
    return sb.length == 0
}

fn (sb: &mut StringBuilder) push_str(s: string) {
    sb.length += len(s)
}

fn (sb: &mut StringBuilder) clear() {
    sb.length = 0
}
