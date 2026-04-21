# EXPECT:
# value 2
# test 7
# b1 5
# b2 10
# b3 1
# 0

class B {
    size Int
}

object A {
    count Int := 2

    # implicit apply declaration
    def (count Int) B = B(size = count)

    # implicit apply declaration
    def (count Int, str String) B = B(size = count)

    # explicit apply declaration
    def apply(str String) B = B(size = 1)

    def value() Int = count

    def test(a Int) Int {
        return a + this.value()
    }
}

def main() Int {

    b1 B = A(5)
    b2 = A(10, "string")
    b3 B = A("string of strings")

    Term.println("value", A.value())
    Term.println("test", A.test(5))
    Term.println("b1", b1.size)
    Term.println("b2", b2.size)
    Term.println("b3", b3.size)
    0
}
