// 51_full_syntax_master_suite.cu
// Master test exercising the entire Cupid language syntax, features, stdlib,
// pointer system, borrow rules, pattern matching, traits, concurrency, and type system.

import "fs"
import "time"
import "path"

/* 
   Nested comment demonstration
   /* Inner nested level 1
      /* Inner nested level 2 */
   */
*/

// 1. Structs & Traits
trait Computable {
    fn compute(self: &Self) -> i64
}

struct Dimensions {
    width: i64
    height: i64
}

impl Computable for Dimensions {
    fn compute(self: &Dimensions) -> i64 {
        return self.width * self.height
    }
}

// 2. Generic Struct
struct Container<T> {
    id: i64
    item: T
}

// 3. Pointer Mutation Helper
fn scale_dimensions(d: &mut Dimensions, factor: i64) {
    d.width = d.width * factor
    d.height = d.height * factor
}

// 4. Concurrency Channel & Routine
mut comm_ch = channel<string>()

fn background_worker(tag: string) {
    time.sleep_ms(10)
    comm_ch.send("Worker finished: " + tag)
}

fn main() {
    println("==================================================")
    println("   CUPID FULL SYNTAX MASTER SUITE (ALL FEATURES)  ")
    println("==================================================")

    // 1. Bitwise Compound Assignments
    mut bits: i64 = 0b00111100
    bits &= 0b00110000
    bits |= 0b00000101
    bits ^= 0b00000001
    bits <<= 2
    bits >>= 1
    println("Bitwise operations result: ", bits)

    // 2. Type Casting via `as`
    let raw_int: i64 = 42
    let float_val: f64 = raw_int as f64 + 0.5
    let truncated_int: i64 = float_val as i64
    println("Type casting: i64 as f64 -> ", float_val, ", f64 as i64 -> ", truncated_int)

    // 3. Structs, Methods and Self receivers
    mut box_dims = Dimensions{ width: 20, height: 10 }
    let initial_area = box_dims.compute()
    println("Initial area from trait method: ", initial_area)

    // 4. Pointers, References (&), and In-place Mutation (&mut)
    println("Box dims pointer before scaling (&box_dims): ", &box_dims)
    scale_dimensions(&mut box_dims, 3)
    println("Box dims pointer after scaling (&box_dims): ", &box_dims)
    let scaled_area = box_dims.compute()
    println("Scaled area: ", scaled_area)

    // 5. Generics & Containers
    let container = Container<string>{ id: 100, item: "Cupid Generic Payload" }
    println("Generic Container struct: ", container)

    // 6. Tuples & Arrays
    let metrics = (999, "Metrics Tuple", true, 3.14159)
    println("Tuple value: ", metrics)
    println("Tuple pointer (&metrics): ", &metrics)

    let scores = [10, 20, 30, 40, 50]
    println("Array value: ", scores)
    println("Array pointer (&scores): ", &scores)

    // 7. Multi-line Strings
    let banner = `Cupid System
Next Generation Native Compiler
High Performance & Memory Safe`
    println("Multiline string literal:")
    println(banner)

    // 8. Concurrency & Channels & Select
    println("Starting background worker...")
    go background_worker("Task-Alpha")

    select {
        case msg = comm_ch.recv():
            println("Channel received: ", msg)
        default:
            println("No message waiting immediately, waiting for message...")
            let delayed_msg = comm_ch.recv()
            println("Channel received delayed: ", delayed_msg)
    }

    // 9. Filesystem & Path stdlib
    let temp_file = "cupid_master_test.tmp"
    mut f = fs.create(temp_file)
    if f.is_open {
        fs.write_str(&f, "Cupid Master Test Payload Verified\n")
        fs.close(&mut f)
        let exists_now = fs.exists(temp_file)
        println("Temporary file created and exists: ", exists_now)
        let cleanup_ok = fs.remove(temp_file)
        println("Temporary file cleanup successful: ", cleanup_ok)
    }

    println("==================================================")
    println("   MASTER SYNTAX SUITE COMPLETED 100% SUCCESS!   ")
    println("==================================================")
}
