// 05_inline_asm.cu: Unsafe block with direct x86-64 assembly

fn main() {
    println("Executing native code with inline assembly:")

    asm {
        mov rax, 100
        add rax, 23
        mov rcx, rax
        call _cupid_print_i64
        call _cupid_println
    }
}
