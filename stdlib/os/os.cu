// Cupid Standard Library: os

export fn exit(code: i32) {
    unsafe {
        asm {
            mov ecx, [rbp - 8]
            call [ExitProcess]
        }
    }
}
