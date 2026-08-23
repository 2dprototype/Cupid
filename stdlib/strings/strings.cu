// Cupid Standard Library: strings

export fn length(s: string) -> i64 {
    return len(s)
}

export fn is_empty(s: string) -> bool {
    return len(s) == 0
}

export fn char_at(s: string, index: i64) -> char {
    return s[index]
}

export fn substring(s: string, start: i64, end: i64) -> string {
    return s[start:end]
}

export fn starts_with(s: string, prefix: string) -> bool {
    let s_len = len(s)
    let p_len = len(prefix)
    if p_len > s_len {
        return false
    }
    let sub = s[0:p_len]
    return sub == prefix
}

export fn ends_with(s: string, suffix: string) -> bool {
    let s_len = len(s)
    let suf_len = len(suffix)
    if suf_len > s_len {
        return false
    }
    let start = s_len - suf_len
    let sub = s[start:s_len]
    return sub == suffix
}

export fn index_of(s: string, substr: string) -> i64 {
    let s_len = len(s)
    let sub_len = len(substr)
    if sub_len == 0 {
        return 0
    }
    if sub_len > s_len {
        return -1
    }
    mut i: i64 = 0
    let limit = s_len - sub_len
    for i <= limit {
        let sub = s[i : i + sub_len]
        if sub == substr {
            return i
        }
        i += 1
    }
    return -1
}

export fn contains(s: string, substr: string) -> bool {
    return index_of(s, substr) >= 0
}

export fn last_index_of(s: string, substr: string) -> i64 {
    let s_len = len(s)
    let sub_len = len(substr)
    if sub_len == 0 {
        return s_len
    }
    if sub_len > s_len {
        return -1
    }
    mut i: i64 = s_len - sub_len
    for i >= 0 {
        let sub = s[i : i + sub_len]
        if sub == substr {
            return i
        }
        i -= 1
    }
    return -1
}

export fn count(s: string, substr: string) -> i64 {
    let s_len = len(s)
    let sub_len = len(substr)
    if sub_len == 0 || sub_len > s_len {
        return 0
    }
    mut cnt: i64 = 0
    mut i: i64 = 0
    let limit = s_len - sub_len
    for i <= limit {
        let sub = s[i : i + sub_len]
        if sub == substr {
            cnt += 1
            i += sub_len
        } else {
            i += 1
        }
    }
    return cnt
}

export fn repeat(s: string, times: i64) -> string {
    if times <= 0 {
        return ""
    }
    mut result = ""
    mut i: i64 = 0
    for i < times {
        result = result + s
        i += 1
    }
    return result
}

export fn to_upper(s: string) -> string {
    let s_len = len(s)
    mut result = ""
    mut i: i64 = 0
    for i < s_len {
        let b = s[i]
        if b >= 'a' && b <= 'z' {
            let up_b = char(i64(b) - 32)
            result = result + string(up_b)
        } else {
            result = result + string(b)
        }
        i += 1
    }
    return result
}

export fn to_lower(s: string) -> string {
    let s_len = len(s)
    mut result = ""
    mut i: i64 = 0
    for i < s_len {
        let b = s[i]
        if b >= 'A' && b <= 'Z' {
            let low_b = char(i64(b) + 32)
            result = result + string(low_b)
        } else {
            result = result + string(b)
        }
        i += 1
    }
    return result
}

export fn replace(s: string, old_str: string, new_str: string) -> string {
    let s_len = len(s)
    let old_len = len(old_str)
    if old_len == 0 || old_len > s_len {
        return s
    }
    mut result = ""
    mut i: i64 = 0
    for i < s_len {
        if i <= s_len - old_len && s[i : i + old_len] == old_str {
            result = result + new_str
            i += old_len
        } else {
            result = result + s[i : i + 1]
            i += 1
        }
    }
    return result
}

export fn trim_space(s: string) -> string {
    let s_len = len(s)
    if s_len == 0 {
        return ""
    }
    mut start: i64 = 0
    for start < s_len {
        let b = s[start]
        if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
            break
        }
        start += 1
    }
    mut end: i64 = s_len
    for end > start {
        let b = s[end - 1]
        if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
            break
        }
        end -= 1
    }
    return s[start:end]
}
