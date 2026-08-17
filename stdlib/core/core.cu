// Cupid Standard Library: core

export trait Comparable {
    fn compare(other: Self) -> i32
}

export trait Equatable {
    fn equals(other: Self) -> bool
}

export trait Clone {
    fn clone() -> Self
}

export trait Copy {
}

export struct Option<T> {
    is_some bool
    value T
}

export fn Some<T>(val: T) -> Option<T> {
    return Option<T>{
        is_some: true
        value: val
    }
}

export fn None<T>() -> Option<T> {
    return Option<T>{
        is_some: false
    }
}

export struct Result<T, E> {
    is_ok bool
    value T
    error E
}

export fn Ok<T, E>(val: T) -> Result<T, E> {
    return Result<T, E>{
        is_ok: true
        value: val
    }
}

export fn Err<T, E>(err: E) -> Result<T, E> {
    return Result<T, E>{
        is_ok: false
        error: err
    }
}
