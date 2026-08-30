// stdlib/time/time.cu: Time and timer utilities for Cupid

// Returns the number of milliseconds that have elapsed since the system was started
export fn now_ticks() -> i64 {
    mut ticks: i64 = 0
    unsafe {
        asm {
            sub rsp, 32
            call [GetTickCount64]
            add rsp, 32
            mov [rbp - 8], rax
        }
    }
    return ticks
}

// Suspends the execution of the current thread for the specified milliseconds
export fn sleep_ms(ms: i64) {
    unsafe {
        asm {
            mov rcx, [rbp - 8]   ; ms
            sub rsp, 32
            call [Sleep]
            add rsp, 32
        }
    }
}

// Calculates elapsed milliseconds since a starting timestamp
export fn elapsed_ms(start: i64) -> i64 {
    let curr = now_ticks()
    if curr >= start {
        return curr - start
    }
    return 0
}
