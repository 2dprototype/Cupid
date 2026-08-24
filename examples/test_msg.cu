// messagebox.cu: Windows MessageBox using inline assembly

fn main() {
    println("Opening Windows MessageBox...")
    
    asm {
        sub rsp, 128
        
        ; Build message string: "Hello from Cupid!"
        mov dword [rsp], 'Hell'
        mov dword [rsp + 4], 'o fr'
        mov dword [rsp + 8], 'om C'
        mov dword [rsp + 12], 'upid'
        mov word [rsp + 16], '!'
        mov byte [rsp + 18], 0
        
        ; Build caption string: "Cupid"
        mov dword [rsp + 32], 'Cupi'
        mov word [rsp + 36], 'd'
        mov byte [rsp + 38], 0
        
        ; MessageBoxA(hWnd, lpText, lpCaption, uType)
        xor rcx, rcx          ; hWnd = NULL
        lea rdx, [rsp]        ; lpText
        lea r8, [rsp + 32]    ; lpCaption
        xor r9, r9            ; uType = MB_OK
        
        ; Call imported MessageBoxA
        call [MessageBoxA]
        
        ; Result is in RAX (1 = IDOK)
        mov rcx, rax
        call _cupid_print_i64
        call _cupid_println
        
        add rsp, 128
    }
    
    println("MessageBox closed!")
}