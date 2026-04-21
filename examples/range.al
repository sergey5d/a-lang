# EXPECT:
# range 1
# range 2
# range 3
# total 6
# another range 10
# another range 11
# another range 12
# another range 13
# another multiplied 17160

def main() Unit {
    total Int := 0
    for item <- Range(1, 4) {
        Term.println("range", item)
        total := total + item
    }
    Term.println("total", total)

    multiplied := 1
    for item <- Range(10, 14) {
        Term.println("another range", item)
        multiplied *= item
    }
    Term.println("another multiplied", multiplied)
}
