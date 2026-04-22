# EXPECT:
# 12
# 14
# 24
# 0

class Adder {
    amount Int

    def apply(value Int) Int = amount + value
}

def main() Int {
    adder Adder = Adder(5)
    Term.println(adder(7))

     lambda Int -> Int = x ->
       adder.apply(x)

     Term.println(lambda(9))

     lambda2 Int -> Int = x -> {
        adder.apply(x)
     }

     Term.println(lambda2(19))
    0
}
