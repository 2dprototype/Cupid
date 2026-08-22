// 04_structs_methods.cu: Struct definitions, instantiation, field access, and methods

struct Player {
    hp i32
    score i32
    alive bool
}

fn (p: &mut Player) damage(amount: i32) {
    p.hp -= amount
    if p.hp <= 0 {
        p.alive = false
    }
}

fn (p: &mut Player) add_score(points: i32) {
    p.score += points
}

fn main() {
    mut p = Player{
        hp: 100
        score: 0
        alive: true
    }

    println("Initial HP:")
    println(p.hp)
    println("Initial alive status:")
    println(p.alive)

    println("Applying 40 damage...")
    p.damage(40)
    println("New HP:")
    println(p.hp)

    println("Adding 150 score...")
    p.add_score(150)
    println("New Score:")
    println(p.score)
}
