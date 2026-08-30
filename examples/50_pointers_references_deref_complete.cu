// 50_pointers_references_deref_complete.cu
// Comprehensive test for pointers, references (&), mutable references (&mut),
// dereferences (*), pointer mutation, struct/tuple references, and pointer formatting in println.

struct Point {
    x: i64
    y: i64
}

struct Node {
    id: i64
    label: string
    active: bool
    weight: f64
}

fn increment_val(num: &mut i64) {
    *num += 10
}

fn update_point(p: &mut Point, new_x: i64, new_y: i64) {
    p.x = new_x
    p.y = new_y
}

fn toggle_node(n: &mut Node) {
    n.active = !n.active
    n.weight = n.weight * 2.0
}

fn main() {
    println("==================================================")
    println("   CUPID POINTERS, REFS & DEREF COMPLETE TEST    ")
    println("==================================================")

    // 1. Primitive reference, dereference and mutation
    mut original_int: i64 = 42
    let ptr_int = &mut original_int
    println("Initial integer pointer (&int): ", ptr_int)
    *ptr_int = 100
    println("Dereferenced after direct store (*ptr = 100): ", *ptr_int)
    increment_val(ptr_int)
    println("After increment_val function (&mut i64): ", original_int)

    // 2. Float pointer & deref mutation
    mut original_float: f64 = 3.14159
    let ptr_float = &mut original_float
    println("Float pointer (&f64): ", ptr_float)
    *ptr_float = 2.71828
    println("Dereferenced float (*ptr_float): ", *ptr_float)

    // 3. Bool pointer & deref mutation
    mut original_bool: bool = true
    let ptr_bool = &mut original_bool
    println("Bool pointer (&bool): ", ptr_bool)
    *ptr_bool = false
    println("Dereferenced bool (*ptr_bool): ", *ptr_bool)

    // 4. Char pointer & deref mutation
    mut original_char: char = 'A'
    let ptr_char = &mut original_char
    println("Char pointer (&char): ", ptr_char)
    *ptr_char = 'Z'
    println("Dereferenced char (*ptr_char): ", *ptr_char)

    // 5. String pointer & deref mutation
    mut original_str: string = "Original text"
    let ptr_str = &mut original_str
    println("String pointer (&string): ", ptr_str)
    *ptr_str = "Mutated text through deref"
    println("Dereferenced string (*ptr_str): ", *ptr_str)

    // 6. Struct references & field mutation
    mut pt = Point{ x: 10, y: 20 }
    println("Struct pointer before update (&pt): ", &pt)
    update_point(&mut pt, 99, 199)
    println("Struct pointer after update_point (&pt): ", &pt)

    mut node = Node{
        id: 777
        label: "Primary Cluster Node"
        active: true
        weight: 12.5
    }
    println("Complex struct pointer (&node): ", &node)
    toggle_node(&mut node)
    println("Complex struct pointer after toggle_node (&node): ", &node)

    // 7. Tuple reference
    let tup = (500, "Tuple in Pointer", true, 88.8)
    let ptr_tup = &tup
    println("Tuple pointer (&tup): ", ptr_tup)

    // 8. Array reference
    let arr = [1, 2, 3, 4, 5]
    let ptr_arr = &arr
    println("Array pointer (&arr): ", ptr_arr)

    println("==================================================")
    println("   POINTERS & REFS TEST COMPLETED SUCCESSFULLY!  ")
    println("==================================================")
}
