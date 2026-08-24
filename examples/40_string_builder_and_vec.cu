// 40_string_builder_and_vec.cu: Ordinary stdlib structs StringBuilder and Vec<T>

import { StringBuilder, new_string_builder } from "strings/builder"
import { Vec, new_vec } from "collections/vec"

fn main() {
    println("=== Testing StringBuilder ===")
    mut sb = new_string_builder()
    sb.push_str("Cupid")
    sb.push_str(" Programming")
    println("StringBuilder length:")
    println(sb.len())
    println("Is empty:")
    println(sb.is_empty())

    println("=== Testing Vec<i64> ===")
    let v: Vec<i64> = new_vec<i64>()
    println("Vec initial length:")
    println(v.len())
    println("Vec is empty:")
    println(v.is_empty())
}
