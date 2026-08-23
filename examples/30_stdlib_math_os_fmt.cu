// 30_stdlib_math_os_fmt.cu: Importing Standard Library modules

import "math"
import "fmt"
import "sync"

fn main() {
    fmt.println_str("Using Cupid standard library:")
    let clamped = math.clamp(50, 0, 100)
    let max_v = math.max(15, 45)
    fmt.println_int(clamped)
    fmt.println_int(max_v)
    sync.sleep_ms(10)
    fmt.println_str("Completed successfully!")
}
