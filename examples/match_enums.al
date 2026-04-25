# EXPECT:
# some 5
# none
# 5
# 100
# left-right
# 0

enum MaybeInt {
    case NoneX
    case SomeX {
        value Int
    }
}

enum PairText {
    case PairX {
        left Str
        right Str
    }
}

def printOption(value MaybeInt) {
    match value {
        SomeX(x) => {
            Term.println("some " + x)
        }
        MaybeInt.NoneX => {
            Term.println("none")
        }
    }
}

def matchSingleLine(value MaybeInt) Int = match value {
    SomeX(x) => x
    MaybeInt.NoneX => 100
}

def main() Int {
    some MaybeInt = MaybeInt.SomeX(5)
    none MaybeInt = MaybeInt.NoneX()
    printOption(some)
    printOption(none)
    Term.println(matchSingleLine(some))
    Term.println(matchSingleLine(none))

    pairText PairText = PairText.PairX("left", "right")
    Term.println(match pairText {
        PairX(left, right) => left + "-" + right
    })

    0
}
