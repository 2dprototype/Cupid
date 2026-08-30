// stdlib/path/path.cu: Path manipulation module for Cupid

import "strings"

// Joins two path components with a directory separator ('/')
export fn join(dir: string, file: string) -> string {
    let dir_len = len(dir)
    if dir_len == 0 {
        return file
    }
    let file_len = len(file)
    if file_len == 0 {
        return dir
    }

    let last_char = dir[dir_len - 1]
    if last_char == '/' || last_char == '\\' {
        return dir + file
    }

    return dir + "/" + file
}

// Returns the last element of path
export fn base(path: string) -> string {
    let l = len(path)
    if l == 0 {
        return ""
    }

    mut last_slash: i64 = -1
    mut i: i64 = 0
    for i < l {
        let ch = path[i]
        if ch == '/' || ch == '\\' {
            last_slash = i
        }
        i += 1
    }

    if last_slash == -1 {
        return path
    }

    return path[last_slash + 1 .. l]
}

// Returns the file extension including the leading dot, e.g. ".cu"
export fn ext(path: string) -> string {
    let l = len(path)
    if l == 0 {
        return ""
    }

    mut last_dot: i64 = -1
    mut i: i64 = l - 1
    for i >= 0 {
        let ch = path[i]
        if ch == '.' {
            last_dot = i
            break
        }
        if ch == '/' || ch == '\\' {
            break
        }
        i -= 1
    }

    if last_dot == -1 {
        return ""
    }

    return path[last_dot .. l]
}

// Returns true if the path is an absolute Windows path (e.g. C:\... or \\...)
export fn is_abs(path: string) -> bool {
    let l = len(path)
    if l < 2 {
        return false
    }

    if path[1] == ':' {
        return true
    }

    if path[0] == '\\' && path[1] == '\\' {
        return true
    }

    if path[0] == '/' {
        return true
    }

    return false
}
