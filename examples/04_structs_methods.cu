// 04_structs_methods.cu: Struct definitions, instantiation, field access, and methods

struct Player {
    hp i32
    score i32
    alive bool
}

impl Player {
    fn damage(mut self, amount: i32) {
        self.hp -= amount
        if self.hp <= 0 {
            self.alive = false
        }
    }
    
    fn add_score(mut self, points: i32) {
        self.score += points
    }
}

fn main() {
    let p = Player{
        hp: 100
        score: 0
        alive: true
    }

    println(p.hp)
    println(p.alive)
}
