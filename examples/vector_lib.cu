// vector_lib.cu: Exported struct and functions for module import testing

export struct Vector {
    x i64
    y i64
}

export fn create_vector(x: i64, y: i64) -> Vector {
    return Vector{
        x: x
        y: y
    }
}

export fn vector_add(v1: Vector, v2: Vector) -> Vector {
    return Vector{
        x: v1.x + v2.x
        y: v1.y + v2.y
    }
}
