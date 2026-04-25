# EXPECT:
# some 5
# none
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

def someInt(value Int) MaybeInt = MaybeInt.SomeX(value)
def noneInt() MaybeInt = MaybeInt.NoneX()

def main() Int {
    some MaybeInt = someInt(5)
    none MaybeInt = noneInt()
    printOption(some)
    printOption(none)

    pair = (1, 2)
    match pair {
        (left, right) => {
            Term.println("tuple " + left + " " + right)
        }
    }

    0
}
