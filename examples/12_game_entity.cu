// 12_game_entity.cu: Game entity simulation with health, position, and damage

struct Entity {
    id i32
    x i32
    y i32
    hp i32
    alive bool
}

fn apply_damage(mut e: Entity, dmg: i32) -> Entity {
    e.hp -= dmg
    if e.hp <= 0 {
        e.hp = 0
        e.alive = false
    }
    return e
}

fn move_entity(mut e: Entity, dx: i32, dy: i32) -> Entity {
    e.x += dx
    e.y += dy
    return e
}

fn main() {
    mut monster = Entity{
        id: 1
        x: 10
        y: 20
        hp: 80
        alive: true
    }

    println("Monster starting HP:")
    println(monster.hp)

    monster = move_entity(monster, 5, -3)
    println("Monster position X:")
    println(monster.x)
    println("Monster position Y:")
    println(monster.y)

    println("Hero strikes monster for 50 damage!")
    monster = apply_damage(monster, 50)
    println("Monster remaining HP:")
    println(monster.hp)
    println("Is monster alive?")
    println(monster.alive)

    println("Hero strikes monster for 40 damage!")
    monster = apply_damage(monster, 40)
    println("Monster remaining HP:")
    println(monster.hp)
    println("Is monster alive?")
    println(monster.alive)
}
