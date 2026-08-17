// 14_modules_import.cu: Testing stdlib imports and local .cu module imports

import "math"
import { abs, max } from "math"
import { Vector, create_vector, vector_add } from "vector_lib.cu"

fn main() {
    println("--- Testing stdlib math module ---")
    let smaller = math.min(15, 42)
    println("math.min(15, 42):")
    println(smaller)

    let larger = max(15, 42)
    println("max(15, 42):")
    println(larger)

    let absolute = abs(-99)
    println("abs(-99):")
    println(absolute)

    println("--- Testing local .cu module import ---")
    let v1 = create_vector(10, 20)
    let v2 = create_vector(5, 15)
    let v3 = vector_add(v1, v2)

    println("Vector 3 X:")
    println(v3.x)
    println("Vector 3 Y:")
    println(v3.y)
}
