// Cupid Standard Library: math

export const PI = 3.141592653589793
export const E = 2.718281828459045

export fn abs(x: i64) -> i64 {
    if x < 0 {
        return -x
    }
    return x
}

export fn min(a: i64, b: i64) -> i64 {
    if a < b {
        return a
    }
    return b
}

export fn max(a: i64, b: i64) -> i64 {
    if a > b {
        return a
    }
    return b
}

export fn clamp(val: i64, low: i64, high: i64) -> i64 {
    if val < low {
        return low
    }
    if val > high {
        return high
    }
    return val
}
