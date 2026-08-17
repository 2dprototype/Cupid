format PE64 console
entry start

include 'C:\Users\Mahdin\Dev\cupid\fasm\INCLUDE\macro\import64.inc'

section '.text' code readable executable

start:
    sub rsp, 40
    mov ecx, -11 ; STD_OUTPUT_HANDLE
    call [GetStdHandle]
    mov [_cupid_stdout], rax

    call cupid_main
    mov ecx, eax
    call [ExitProcess]

_cupid_print_str:
    push rbp
    mov rbp, rsp
    sub rsp, 48
    mov rdx, rcx
    lea rcx, [_cupid_fmt_str]
    call [printf]
    mov rsp, rbp
    pop rbp
    ret

_cupid_print_i64:
    push rbp
    mov rbp, rsp
    sub rsp, 48
    mov rdx, rcx
    lea rcx, [_cupid_fmt_i64]
    call [printf]
    mov rsp, rbp
    pop rbp
    ret

_cupid_print_bool:
    push rbp
    mov rbp, rsp
    sub rsp, 48
    cmp rcx, 0
    je .print_false
    lea rdx, [_cupid_true_str]
    jmp .do_print_bool
.print_false:
    lea rdx, [_cupid_false_str]
.do_print_bool:
    lea rcx, [_cupid_fmt_str]
    call [printf]
    mov rsp, rbp
    pop rbp
    ret

_cupid_println:
    push rbp
    mov rbp, rsp
    sub rsp, 48
    lea rdx, [_cupid_crlf]
    lea rcx, [_cupid_fmt_str]
    call [printf]
    mov rsp, rbp
    pop rbp
    ret

_cupid_alloc:
    push rbp
    mov rbp, rsp
    sub rsp, 48
    mov r8, rcx ; bytes
    call [GetProcessHeap]
    mov rcx, rax ; hHeap
    mov edx, 8   ; HEAP_ZERO_MEMORY
    call [HeapAlloc]
    mov rsp, rbp
    pop rbp
    ret

_cupid_free:
    push rbp
    mov rbp, rsp
    sub rsp, 48
    mov r8, rcx ; lpMem
    call [GetProcessHeap]
    mov rcx, rax ; hHeap
    xor edx, edx ; dwFlags
    call [HeapFree]
    mov rsp, rbp
    pop rbp
    ret

cu_vec2_add:
    push rbp
    mov rbp, rsp
    sub rsp, 144
    mov rax, [rcx + 0]
    mov [rbp - 16 + 0], rax
    mov rax, [rcx + 8]
    mov [rbp - 16 + 8], rax
    mov rax, [rdx + 0]
    mov [rbp - 32 + 0], rax
    mov rax, [rdx + 8]
    mov [rbp - 32 + 8], rax
.bb_cu_vec2_add_0:
    mov rcx, [rbp - 16 + 0]
    mov [rbp - 56], rcx
    mov rcx, [rbp - 32 + 0]
    mov [rbp - 64], rcx
    mov rax, [rbp - 56]
    mov rcx, [rbp - 64]
    add rax, rcx
    mov [rbp - 72], rax
    mov rcx, [rbp - 72]
    mov [rbp - 48 + 0], rcx
    mov rcx, [rbp - 16 + 8]
    mov [rbp - 80], rcx
    mov rcx, [rbp - 32 + 8]
    mov [rbp - 88], rcx
    mov rax, [rbp - 80]
    mov rcx, [rbp - 88]
    add rax, rcx
    mov [rbp - 96], rax
    mov rcx, [rbp - 96]
    mov [rbp - 48 + 8], rcx
    lea rax, [rbp - 48]
    jmp .epilogue_cu_vec2_add
.epilogue_cu_vec2_add:
    mov rsp, rbp
    pop rbp
    ret

cu_vec2_dot:
    push rbp
    mov rbp, rsp
    sub rsp, 144
    mov rax, [rcx + 0]
    mov [rbp - 16 + 0], rax
    mov rax, [rcx + 8]
    mov [rbp - 16 + 8], rax
    mov rax, [rdx + 0]
    mov [rbp - 32 + 0], rax
    mov rax, [rdx + 8]
    mov [rbp - 32 + 8], rax
.bb_cu_vec2_dot_0:
    mov rcx, [rbp - 16 + 0]
    mov [rbp - 40], rcx
    mov rcx, [rbp - 32 + 0]
    mov [rbp - 48], rcx
    mov rax, [rbp - 40]
    mov rcx, [rbp - 48]
    imul rax, rcx
    mov [rbp - 56], rax
    mov rcx, [rbp - 16 + 8]
    mov [rbp - 64], rcx
    mov rcx, [rbp - 32 + 8]
    mov [rbp - 72], rcx
    mov rax, [rbp - 64]
    mov rcx, [rbp - 72]
    imul rax, rcx
    mov [rbp - 80], rax
    mov rax, [rbp - 56]
    mov rcx, [rbp - 80]
    add rax, rcx
    mov [rbp - 88], rax
    mov rax, [rbp - 88]
    jmp .epilogue_cu_vec2_dot
.epilogue_cu_vec2_dot:
    mov rsp, rbp
    pop rbp
    ret

cu_vec2_length_squared:
    push rbp
    mov rbp, rsp
    sub rsp, 128
    mov rax, [rcx + 0]
    mov [rbp - 16 + 0], rax
    mov rax, [rcx + 8]
    mov [rbp - 16 + 8], rax
.bb_cu_vec2_length_squared_0:
    mov rcx, [rbp - 16 + 0]
    mov [rbp - 24], rcx
    mov rcx, [rbp - 16 + 0]
    mov [rbp - 32], rcx
    mov rax, [rbp - 24]
    mov rcx, [rbp - 32]
    imul rax, rcx
    mov [rbp - 40], rax
    mov rcx, [rbp - 16 + 8]
    mov [rbp - 48], rcx
    mov rcx, [rbp - 16 + 8]
    mov [rbp - 56], rcx
    mov rax, [rbp - 48]
    mov rcx, [rbp - 56]
    imul rax, rcx
    mov [rbp - 64], rax
    mov rax, [rbp - 40]
    mov rcx, [rbp - 64]
    add rax, rcx
    mov [rbp - 72], rax
    mov rax, [rbp - 72]
    jmp .epilogue_cu_vec2_length_squared
.epilogue_cu_vec2_length_squared:
    mov rsp, rbp
    pop rbp
    ret

cupid_main:
    push rbp
    mov rbp, rsp
    sub rsp, 256
.bb_cupid_main_0:
    mov rcx, 3
    mov [rbp - 16 + 0], rcx
    mov rcx, 4
    mov [rbp - 16 + 8], rcx
    mov rax, [rbp - 16 + 0]
    mov [rbp - 32 + 0], rax
    mov rax, [rbp - 16 + 8]
    mov [rbp - 32 + 8], rax
    mov rcx, 1
    mov [rbp - 48 + 0], rcx
    mov rcx, 2
    mov [rbp - 48 + 8], rcx
    mov rax, [rbp - 48 + 0]
    mov [rbp - 64 + 0], rax
    mov rax, [rbp - 48 + 8]
    mov [rbp - 64 + 8], rax
    lea rcx, [rbp - 32]
    lea rdx, [rbp - 64]
    call cu_vec2_add
    mov rcx, [rax + 0]
    mov [rbp - 80 + 0], rcx
    mov rcx, [rax + 8]
    mov [rbp - 80 + 8], rcx
    mov rax, [rbp - 80 + 0]
    mov [rbp - 96 + 0], rax
    mov rax, [rbp - 80 + 8]
    mov [rbp - 96 + 8], rax
    lea rcx, [_cupid_str_1]
    call _cupid_print_str
    call _cupid_println
    mov rcx, [rbp - 96 + 0]
    mov [rbp - 112], rcx
    mov rcx, [rbp - 112]
    call _cupid_print_i64
    call _cupid_println
    lea rcx, [_cupid_str_2]
    call _cupid_print_str
    call _cupid_println
    mov rcx, [rbp - 96 + 8]
    mov [rbp - 136], rcx
    mov rcx, [rbp - 136]
    call _cupid_print_i64
    call _cupid_println
    lea rcx, [rbp - 32]
    lea rdx, [rbp - 64]
    call cu_vec2_dot
    mov [rbp - 152], rax
    mov rax, [rbp - 152]
    mov [rbp - 160], rax
    lea rcx, [_cupid_str_3]
    call _cupid_print_str
    call _cupid_println
    mov rcx, [rbp - 160]
    call _cupid_print_i64
    call _cupid_println
    lea rcx, [rbp - 32]
    call cu_vec2_length_squared
    mov [rbp - 184], rax
    mov rax, [rbp - 184]
    mov [rbp - 192], rax
    lea rcx, [_cupid_str_4]
    call _cupid_print_str
    call _cupid_println
    mov rcx, [rbp - 192]
    call _cupid_print_i64
    call _cupid_println
    jmp .epilogue_cupid_main
.epilogue_cupid_main:
    mov rsp, rbp
    pop rbp
    ret

section '.data' data readable writeable
    _cupid_stdout dq 0
    _cupid_bytes_written dq 0
    _cupid_crlf db 13, 10, 0
    _cupid_true_str db 'true', 0
    _cupid_false_str db 'false', 0
    _cupid_fmt_i64 db '%lld', 0
    _cupid_fmt_str db '%s', 0
    _cupid_str_3 db 'Dot product:', 0
    _cupid_str_3_len dq 12
    _cupid_str_4 db 'Length squared of (3,4):', 0
    _cupid_str_4_len dq 24
    _cupid_str_1 db 'Vector sum x:', 0
    _cupid_str_1_len dq 13
    _cupid_str_2 db 'Vector sum y:', 0
    _cupid_str_2_len dq 13

section '.idata' import data readable writeable
library kernel32,'KERNEL32.DLL', \
        msvcrt,'MSVCRT.DLL'

import kernel32, \
       ExitProcess,'ExitProcess', \
       GetStdHandle,'GetStdHandle', \
       WriteFile,'WriteFile', \
       ReadFile,'ReadFile', \
       GetProcessHeap,'GetProcessHeap', \
       HeapAlloc,'HeapAlloc', \
       HeapFree,'HeapFree', \
       Sleep,'Sleep'

import msvcrt, \
       printf,'printf'
