// Cupid Standard Library: collections/vector
// Dynamic generic array written in pure Cupid

import { Option, Some, None } from "core"

export struct Vector<T> {
    data: []T
    length: i64
    capacity: i64
}

export fn new_vector<T>() -> Vector<T> {
    return Vector<T>{
        length: 0
        capacity: 0
    }
}

fn (v: &Vector<T>) len() -> i64 {
    return v.length
}

fn (v: &Vector<T>) is_empty() -> bool {
    return v.length == 0
}

fn (v: &Vector<T>) get(index: i64) -> Option<T> {
    if index < 0 || index >= v.length {
        return None<T>()
    }
    return Some<T>(v.data[index])
}

fn (v: &mut Vector<T>) clear() {
    v.length = 0
}
