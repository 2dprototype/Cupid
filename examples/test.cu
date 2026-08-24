// struct User {
    // name string
    // age i8
// }

// fn main() {
    // let arr: [2]User = [User{name:"Alua"}, User{name:"Aluax"}]
    
    // println(arr)
    // println(typeof(arr))
    // println(len(arr))
// }

fn main() {
    // let arr: []u8 = [1,2,3,4,5,6]
    
    // println(arr)
    // println(len(arr))
    mut str = "Hello"
    mut b = &str
    *b = "U"
    println(str)
    println(str[0])
}