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
        mov rcx, [rbp - 8]       ; path string pointer
        mov rdx, 80000000h       ; dwDesiredAccess (GENERIC_READ)
        mov r8, 1                ; dwShareMode (FILE_SHARE_READ)
        mov r9, 0                ; lpSecurityAttributes (NULL)
        sub rsp, 32 + 24         ; shadow space + 3 extra stack parameters
        mov qword [rsp + 32], 3  ; dwCreationDisposition (OPEN_EXISTING)
        mov qword [rsp + 40], 80h; dwFlagsAndAttributes (FILE_ATTRIBUTE_NORMAL)
        mov qword [rsp + 48], 0  ; hTemplateFile (NULL)
        call [CreateFileA]
        add rsp, 32 + 24
        mov [rbp - 16], rax      ; store handle into file_handle
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
        mov rcx, [rbp - 8]       ; path string pointer
        mov rdx, 40000000h       ; dwDesiredAccess (GENERIC_WRITE)
        mov r8, 0                ; dwShareMode (0)
        mov r9, 0                ; lpSecurityAttributes (NULL)
        sub rsp, 32 + 24
        mov qword [rsp + 32], 2  ; dwCreationDisposition (CREATE_ALWAYS)
        mov qword [rsp + 40], 80h; dwFlagsAndAttributes (FILE_ATTRIBUTE_NORMAL)
        mov qword [rsp + 48], 0  ; hTemplateFile (NULL)
        call [CreateFileA]
        add rsp, 32 + 24
        mov [rbp - 16], rax      ; store handle into file_handle
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
    if !f.is_open || f.handle == 0 {
        return false
    }

    let text_len: i64 = len(text)
    mut bytes_written: i64 = 0

    asm {
        mov rcx, [rbp - 8]       ; &File
        mov rcx, [rcx]           ; f.handle
        mov rdx, [rbp - 16]      ; text string pointer
        mov r8, [rbp - 24]       ; text_len
        lea r9, [rbp - 32]       ; &bytes_written
        sub rsp, 32 + 8
        mov qword [rsp + 32], 0  ; lpOverlapped (NULL)
        call [WriteFile]
        add rsp, 32 + 8
    }

    return bytes_written == text_len
}

// Closes an open file
export fn close(f: &mut File) {
    if f.is_open && f.handle != 0 {
        asm {
            mov rcx, [rbp - 8]   ; &mut File
            mov rcx, [rcx]       ; f.handle
            sub rsp, 32
            call [CloseHandle]
            add rsp, 32
        }
        f.handle = 0
        f.is_open = false
    }
}

// Checks whether a file exists
export fn exists(path: string) -> bool {
    mut attr: i64 = 0
    asm {
        mov rcx, [rbp - 8]       ; path string pointer
        sub rsp, 32
        call [GetFileAttributesA]
        add rsp, 32
        mov [rbp - 16], rax      ; store return code into attr
    }
    // INVALID_FILE_ATTRIBUTES = -1
    return attr != -1
}

// Deletes a file
export fn remove(path: string) -> bool {
    mut success: i64 = 0
    asm {
        mov rcx, [rbp - 8]       ; path string pointer
        sub rsp, 32
        call [DeleteFileA]
        add rsp, 32
        mov [rbp - 16], rax      ; store result into success
    }
    return success != 0
}
