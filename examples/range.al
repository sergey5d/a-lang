# EXPECT:
# range 1
# range 2
# range 3
# total 6
# compact range 10
# compact range 14
# compact multiplied 140

def main() Unit {
    total Int := 0
    for item <- Range(1, 4) {
        Term.println("range", item)
        total := total + item
    }
    Term.println("total", total)

    multiplied := 1
    for item <- [10...14] {
        Term.println("compact range", item)
        multiplied *= item
    }
    Term.println("compact multiplied", multiplied)
}
