# EXPECT:
# 12
# 14

class Adder {
    amount Int

    def apply(value Int) Int = amount + value
}

def main() Int {
    adder Adder = Adder(5)
    Term.println(adder(7))

    lambda Int -> Int = adder.apply

    Term.println(lambda(9))
}
