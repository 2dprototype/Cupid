// Cupid Standard Library: math

export const PI = 3.141592653589793
export const E = 2.718281828459045

export fn abs(x: i32) -> i32 {
    if x < 0 {
        return -x
    }
    return x
}

export fn min(a: i32, b: i32) -> i32 {
    if a < b {
        return a
    }
    return b
}

export fn max(a: i32, b: i32) -> i32 {
    if a > b {
        return a
    }
    return b
}

export fn clamp(val: i32, low: i32, high: i32) -> i32 {
    if val < low {
        return low
    }
    if val > high {
        return high
    }
    return val
}
