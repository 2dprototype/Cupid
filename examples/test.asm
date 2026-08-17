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

cupid_main:
    push rbp
    mov rbp, rsp
    sub rsp, 80
.bb_main_0:
    mov rax, [global_10.1]
    mov [rbp - 8], rax
    mov rax, [rbp - 8]
    mov rcx, [global_2.1]
    imul rax, rcx
    mov [rbp - 16], rax
    mov rcx, [rbp - 16]
    call _cupid_print_i64
    call _cupid_println
    jmp .epilogue_main
.epilogue_main:
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
