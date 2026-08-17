// 11_temperature.cu: Temperature scale conversions

fn celsius_to_fahrenheit(celsius: i64) -> i64 {
    return (celsius * 9 / 5) + 32
}

fn fahrenheit_to_celsius(fahrenheit: i64) -> i64 {
    return (fahrenheit - 32) * 5 / 9
}

fn celsius_to_kelvin(celsius: i64) -> i64 {
    return celsius + 273
}

fn main() {
    let water_freezing_c = 0
    let water_boiling_c = 100
    let room_temp_c = 25

    println("Water freezing in F:")
    println(celsius_to_fahrenheit(water_freezing_c))

    println("Water boiling in F:")
    println(celsius_to_fahrenheit(water_boiling_c))

    println("Room temp in K:")
    println(celsius_to_kelvin(room_temp_c))

    println("77F converted back to C:")
    println(fahrenheit_to_celsius(77))
}
