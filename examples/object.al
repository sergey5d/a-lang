# EXPECT:
# value 2
# test 7
# b1 5
# b2 1
# 0

class B {
    size Int
}

object A {
    count Int := 2

    # explicit apply declaration
    def apply(count Int) B = B(size = count)

    # explicit apply declaration
    def apply(str Str) B = B(size = 1)

    def value() Int = count

    def test(a Int) Int {
        return a + this.value()
    }
}

def main() Int {

    b1 B = A.apply(5)
    b2 B = A.apply("string of strings")

    OS.println("value", A.value())
    OS.println("test", A.test(5))
    OS.println("b1", b1.size)
    OS.println("b2", b2.size)
    0
}
