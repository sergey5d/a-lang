# EXPECT:
# 10
# description
# 5
# 100 description description
# 0
# 101 description description updated
# 0
# 102 description description updated
# 7
# Hop-hop
# Jump 3 steps

# records are read only and do not allow mutable fields

interface Hopper {
    def hop() Str
}

interface Jumper {
    def jump(steps Int) Str
}

record Amount with Hopper, Jumper {
    amount Int
    description Str
    count Int

    def multiple(other Amount) Amount = Amount(
        amount = this.amount * other.amount,
        description = this.description + " " + other.description,
        count = 0
    )

    def reportAmount() {
        Term.println("amount:", amount)
    }

    impl def hop() Str = "Hop-hop"

    impl def jump(steps Int) Str = "Jump " + steps + " steps"
}

a1 = Amount(10, "description", 5)

a2 = a1.multiple(a1)

amountAmount = a1.amount
amountDescr = a1.description

a3 = a2 with { amount = 101, description = a2.description + " updated" }

a4 = a3 with { amount = 102 } with { count = 7 }

def main() Unit {
    Term.println(amountAmount)
    Term.println(amountDescr)
    Term.println(a1.count)
    Term.println(a2.amount, a2.description)
    Term.println(a2.count)
    Term.println(a3.amount, a3.description)
    Term.println(a3.count)
    Term.println(a4.amount, a4.description)
    Term.println(a4.count)
    Term.println(a1.hop())
    Term.println(a1.jump(3))
}
