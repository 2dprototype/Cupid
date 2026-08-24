// Cupid Standard Library: collections/vec
// Ordinary generic struct for growable arrays

import { Option, Some, None } from "core"

export struct Vec<T> {
    data: []T
    length: i64
    capacity: i64
}

export fn new_vec<T>() -> Vec<T> {
    return Vec<T>{
        length: 0
        capacity: 0
    }
}

fn (v: &Vec<T>) len() -> i64 {
    return v.length
}

fn (v: &Vec<T>) is_empty() -> bool {
    return v.length == 0
}

fn (v: &Vec<T>) get(index: i64) -> Option<T> {
    if index < 0 || index >= v.length {
        return None<T>()
    }
    return Some<T>(v.data[index])
}

fn (v: &mut Vec<T>) clear() {
    v.length = 0
}
