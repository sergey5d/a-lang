# EXPECT:
# some 5
# 5
# 0

enum MaybeInt {
    case NoneX
    case SomeX {
        value Int
    }
}

def main() Int {
    maybe MaybeInt = MaybeInt.SomeX(5)

    match maybe: SomeX(x) => Term.println("some " + x), MaybeInt.NoneX => Term.println("none")

    picked = match maybe: SomeX(x) => x, MaybeInt.NoneX => 100
    Term.println(picked)

    0
}
