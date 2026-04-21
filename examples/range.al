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
# descending 5
# descending 4
# descending 3
# descending 2
# descending total 14
# stepped 10
# stepped 8
# stepped 6
# stepped total 24

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

    descendingTotal := 0
    for item <- Range(5, 1) {
        Term.println("descending", item)
        descendingTotal += item
    }
    Term.println("descending total", descendingTotal)

    steppedTotal := 0
    for item <- Range(10, 4, -2) {
        Term.println("stepped", item)
        steppedTotal += item
    }
    Term.println("stepped total", steppedTotal)
}
