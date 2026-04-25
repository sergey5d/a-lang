# EXPECT:
# pair 5-9
# 14
# 0

class PairBox {
    left Int
    right Int
}

def main() Int {
    pair PairBox = PairBox(5, 9)

    match pair: PairBox(left, right) => Term.println("pair " + left + "-" + right)

    picked = match pair: PairBox(left, right) => left + right
    Term.println(picked)

    0
}
