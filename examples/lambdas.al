# EXPECT:
# 14
# 24
# 34
# 45
# 56
# 66
# 0

class Adder {
    amount Int

    def apply(value Int) Int = amount + value
}

def main() Int {
    adder Adder = Adder(5)

    lambda Int -> Int = x ->
       adder.apply(x)

    OS.println(lambda(9))

    lambda2 = (x Int) -> adder.apply(x)
    OS.println(lambda2(19))

    lambda3 Int -> Int = x -> {
        adder.apply(x)
    }
    OS.println(lambda3(29))

    lambda4 = x Int -> {
        adder.apply(x)
    }
    OS.println(lambda4(40))

    lambda5 (Int, Int) -> Int = (left, right) ->
        adder.apply(left + right)
    OS.println(lambda5(20, 31))

    lambda6 = (left Int, right Int) -> adder.apply(left + right)
    OS.println(lambda6(30, 31))
    0
}
