// 32_receiver_methods_and_borrowing.cu: Testing Go-style receivers and borrow modes in Cupid
// No 'self' keyword - purely explicit named receivers with static types and borrow specifiers.

struct Rectangle {
    width: i64
    height: i64
}

// 1. Value receiver (copies/moves value)
fn (r: Rectangle) area() -> i64 {
    return r.width * r.height
}

// 2. Immutable borrow receiver (read-only reference)
fn (r: &Rectangle) perimeter() -> i64 {
    return (r.width + r.height) * 2
}

// 3. Mutable borrow receiver (modifies the instance in place)
fn (r: &mut Rectangle) scale(factor: i64) {
    r.width *= factor
    r.height *= factor
}

// 4. Another struct with receiver methods
struct BankAccount {
    account_id: i64
    balance: i64
}

fn (acc: &BankAccount) get_balance() -> i64 {
    return acc.balance
}

fn (acc: &mut BankAccount) deposit(amount: i64) {
    acc.balance += amount
}

fn (acc: &mut BankAccount) withdraw(amount: i64) -> bool {
    if acc.balance >= amount {
        acc.balance -= amount
        return true
    }
    return false
}

fn main() {
    println("=== Testing Rectangle Receiver Methods ===")
    mut rect = Rectangle{ width: 10, height: 5 }

    println("Initial Area (10 * 5 = 50):")
    println(rect.area())

    println("Initial Perimeter (2 * (10 + 5) = 30):")
    println(rect.perimeter())

    println("Scaling rectangle by 3...")
    rect.scale(3)

    println("New Width (30):")
    println(rect.width)
    println("New Height (15):")
    println(rect.height)
    println("Scaled Area (450):")
    println(rect.area())

    println("=== Testing BankAccount Receiver Methods ===")
    mut my_acc = BankAccount{ account_id: 1001, balance: 500 }
    println("Current Balance:")
    println(my_acc.get_balance())

    println("Depositing 250...")
    my_acc.deposit(250)
    println("New Balance (750):")
    println(my_acc.get_balance())

    println("Withdrawing 1000 (should fail):")
    let ok1 = my_acc.withdraw(1000)
    println(ok1)

    println("Withdrawing 200 (should succeed):")
    let ok2 = my_acc.withdraw(200)
    println(ok2)
    println("Final Balance (550):")
    println(my_acc.get_balance())
}
