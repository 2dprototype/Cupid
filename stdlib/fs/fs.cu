// stdlib/fs/fs.cu: Native filesystem module for Cupid

export struct File {
    handle: i64
    is_open: bool
}

// Opens an existing file for reading
export fn open_read(path: string) -> File {
    mut file_handle: i64 = 0
    asm {
        ; GENERIC_READ = 0x80000000
        ; FILE_SHARE_READ = 1
        ; OPEN_EXISTING = 3
        ; FILE_ATTRIBUTE_NORMAL = 0x80
        mov rcx, [rbp - 16]      ; path string pointer
        mov rdx, 80000000h       ; dwDesiredAccess (GENERIC_READ)
        mov r8, 1                ; dwShareMode (FILE_SHARE_READ)
        mov r9, 0                ; lpSecurityAttributes (NULL)
        sub rsp, 64              ; 32 shadow + 24 args + 8 alignment = 64 (16-byte aligned)
        mov qword [rsp + 32], 3  ; dwCreationDisposition (OPEN_EXISTING)
        mov qword [rsp + 40], 80h; dwFlagsAndAttributes (FILE_ATTRIBUTE_NORMAL)
        mov qword [rsp + 48], 0  ; hTemplateFile (NULL)
        call [CreateFileA]
        add rsp, 64
        mov [rbp - 24], rax      ; store handle into file_handle
    }

    if file_handle == -1 || file_handle == 0 {
        return File{
            handle: 0
            is_open: false
        }
    }

    return File{
        handle: file_handle
        is_open: true
    }
}

// Creates or truncates a file for writing
export fn create(path: string) -> File {
    mut file_handle: i64 = 0
    asm {
        ; GENERIC_WRITE = 0x40000000
        ; CREATE_ALWAYS = 2
        ; FILE_ATTRIBUTE_NORMAL = 0x80
        mov rcx, [rbp - 16]      ; path string pointer
        mov rdx, 40000000h       ; dwDesiredAccess (GENERIC_WRITE)
        mov r8, 0                ; dwShareMode (0)
        mov r9, 0                ; lpSecurityAttributes (NULL)
        sub rsp, 64              ; 32 shadow + 24 args + 8 alignment = 64 (16-byte aligned)
        mov qword [rsp + 32], 2  ; dwCreationDisposition (CREATE_ALWAYS)
        mov qword [rsp + 40], 80h; dwFlagsAndAttributes (FILE_ATTRIBUTE_NORMAL)
        mov qword [rsp + 48], 0  ; hTemplateFile (NULL)
        call [CreateFileA]
        add rsp, 64
        mov [rbp - 24], rax      ; store handle into file_handle
    }

    if file_handle == -1 || file_handle == 0 {
        return File{
            handle: 0
            is_open: false
        }
    }

    return File{
        handle: file_handle
        is_open: true
    }
}

// Writes text to an open file
export fn write_str(f: &File, text: string) -> bool {
    mut success: i64 = 0
    asm {
        mov rcx, [rbp - 8]       ; &File pointer
        mov al, byte [rcx + 8]   ; f.is_open
        cmp al, 0
        je .ws_fail
        mov rcx, [rcx + 0]       ; f.handle
        cmp rcx, 0
        je .ws_fail

        mov rdx, [rbp - 24]      ; text string data pointer
        mov r8, [rbp - 16]       ; text string length
        sub rsp, 64              ; 32 shadow + 8 arg + 24 local/align = 64 (16-byte aligned)
        lea r9, [rsp + 40]       ; temporary slot for lpNumberOfBytesWritten
        mov qword [rsp + 32], 0  ; lpOverlapped (NULL)
        call [WriteFile]
        add rsp, 64
        mov [rbp - 32], rax      ; store result into success
        jmp .ws_done
    .ws_fail:
        mov qword [rbp - 32], 0
    .ws_done:
    }

    return success != 0
}

// Closes an open file
export fn close(f: &mut File) {
    asm {
        mov rcx, [rbp - 8]       ; &mut File
        mov al, byte [rcx + 8]   ; f.is_open
        cmp al, 0
        je .close_done
        mov rcx, [rcx + 0]       ; f.handle
        cmp rcx, 0
        je .close_done
        sub rsp, 32              ; shadow space (16-byte aligned)
        call [CloseHandle]
        add rsp, 32
    .close_done:
    }
    f.handle = 0
    f.is_open = false
}

// Checks whether a file exists
export fn exists(path: string) -> bool {
    mut attr: i64 = 0
    asm {
        mov rcx, [rbp - 16]      ; path string pointer
        sub rsp, 32              ; shadow space (16-byte aligned)
        call [GetFileAttributesA]
        add rsp, 32
        mov [rbp - 24], rax      ; store return code into attr
    }
    // INVALID_FILE_ATTRIBUTES = -1 (0xFFFFFFFF in 32-bit or -1 in 64-bit)
    return attr != -1 && attr != 4294967295
}

// Deletes a file
export fn remove(path: string) -> bool {
    mut success: i64 = 0
    asm {
        mov rcx, [rbp - 16]      ; path string pointer
        sub rsp, 32              ; shadow space (16-byte aligned)
        call [DeleteFileA]
        add rsp, 32
        mov [rbp - 24], rax      ; store result into success
    }
    return success != 0
}
