import "fs" 
 
fn main() {
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
}