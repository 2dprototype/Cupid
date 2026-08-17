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
