# EXPECT:
# 10
# description
# 100 description description
# Hop-hop
# Jump 3 steps

# records are read only and do not allow mutable fields

interface Hopper {
    def hop() String
}

interface Jumper {
    def jump(steps Int) String
}

record Amount with Hopper, Jumper {
    amount Int
    description String

    def multiple(other Amount) Amount = Amount(
        amount = this.amount * other.amount,
        description = this.description + " " + other.description
    )

    def reportAmount() {
        Term.println("amount:", amount)
    }

    def hop() String = "Hop-hop"

    def jump(steps Int) String = "Jump " + steps + " steps"
}

a1 = Amount(10, "description")

a2 = a1.multiple(a1)

amountAmount = a1.amount
amountDescr = a1.description

def main() Unit {
    Term.println(amountAmount)
    Term.println(amountDescr)
    Term.println(a2.amount, a2.description)
    Term.println(a1.hop())
    Term.println(a1.jump(3))
}
