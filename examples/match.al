# EXPECT:
# some 5
# none
# 5
# 100
# tuple 1 2
# 0

enum MaybeInt {
    case NoneX
    case SomeX {
        value Int
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

def someInt(value Int) MaybeInt = MaybeInt.SomeX(value)
def noneInt() MaybeInt = MaybeInt.NoneX()

def main() Int {
    some MaybeInt = someInt(5)
    none MaybeInt = noneInt()
    printOption(some)
    printOption(none)
    Term.println(matchSingleLine(some))
    Term.println(matchSingleLine(none))

    pair = (1, 2)
    match pair {
        (left, right) => {
            Term.println("tuple " + left + " " + right)
        }
    }

    0
}
