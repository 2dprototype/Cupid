// 33_traits_and_generic_contracts.cu: Testing trait declarations and implementations without 'self' or 'Self'
// Everything uses explicit types and generic parameters <T>.

trait Summable<T> {
    fn add_to(target: &mut T, value: i64)
}

struct ScoreBoard {
    total_score: i64
    player_count: i64
}

fn (sb: &mut ScoreBoard) record_score(points: i64) {
    sb.total_score += points
    sb.player_count += 1
}

fn (sb: &ScoreBoard) average() -> i64 {
    if sb.player_count == 0 {
        return 0
    }
    return sb.total_score / sb.player_count
}

fn main() {
    println("=== Testing ScoreBoard ===")
    mut board = ScoreBoard{ total_score: 0, player_count: 0 }

    println("Recording player 1 score: 100")
    board.record_score(100)

    println("Recording player 2 score: 200")
    board.record_score(200)

    println("Recording player 3 score: 150")
    board.record_score(150)

    println("Total Score (450):")
    println(board.total_score)

    println("Player Count (3):")
    println(board.player_count)

    println("Average Score (150):")
    println(board.average())
}
