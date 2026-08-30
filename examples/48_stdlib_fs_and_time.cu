// 48_stdlib_fs_and_time.cu
// Demonstrates native file I/O (stdlib/fs) and time measuring (stdlib/time)

import "fs"
import "time"
import "path"

fn main() {
    println("==================================================")
    println("   CUPID STDLIB: FS, PATH, AND TIME DEMO          ")
    println("==================================================")

    // Path operations
    let full_path = path.join("C:/Temp", "cupid_test.txt")
    println("Joined path:")
    println(full_path)

    let file_base = path.base(full_path)
    println("Base filename:")
    println(file_base)

    let file_ext = path.ext(full_path)
    println("File extension:")
    println(file_ext)

    // Time operations
    let start_time = time.now_ticks()
    println("Starting timer ticks:")
    println(start_time)

    println("Sleeping for 15ms...")
    time.sleep_ms(15)

    let elapsed = time.elapsed_ms(start_time)
    println("Elapsed ms:")
    println(elapsed)

    // File writing & checking
    let test_filename = "cupid_temp_output.txt"
    println("Creating file...")
    mut f = fs.create(test_filename)
    if f.is_open {
        println("File opened successfully for writing.")
        fs.write_str(&f, "Hello from Cupid native stdlib/fs!\nSecond line of Cupid text.\n")
        fs.close(&mut f)
        println("File written and closed.")
    } 
    else {
        println("Could not create file.")
    }

    let file_exists = fs.exists(test_filename)
    println("Checking if file exists:")
    println(file_exists)

    // Clean up temporary file
    let removed = fs.remove(test_filename)
    println("File cleanup removed:")
    println(removed)

    println("==================================================")
    println("   STDLIB DEMO COMPLETED SUCCESSFULLY!            ")
    println("==================================================")
}
