// 10_banking_system.cu: Struct-based banking simulation

struct BankAccount {
    account_id i32
    balance i32
    active bool
}

fn deposit(mut acc: BankAccount, amount: i32) -> BankAccount {
    if acc.active && amount > 0 {
        acc.balance += amount
        println("Deposited amount:")
        println(amount)
    }
    return acc
}

fn withdraw(mut acc: BankAccount, amount: i32) -> BankAccount {
    if acc.active && amount > 0 && acc.balance >= amount {
        acc.balance -= amount
        println("Withdrew amount:")
        println(amount)
    } else {
        println("Insufficient funds or invalid account.")
    }
    return acc
}

fn main() {
    mut my_acc = BankAccount{
        account_id: 1001
        balance: 500
        active: true
    }

    println("Initial Balance:")
    println(my_acc.balance)

    my_acc = deposit(my_acc, 250)
    println("New Balance:")
    println(my_acc.balance)

    my_acc = withdraw(my_acc, 100)
    println("Final Balance:")
    println(my_acc.balance)

    my_acc = withdraw(my_acc, 1000)
}
