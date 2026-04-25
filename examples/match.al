# EXPECT:
# some 5
# none
# 5
# 100
# exact tuple
# exact tuple 2-3
# exact tuple 4-5
# exact value
# 7
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

interface WorkerLike {
    def doWork() Int
}

class Worker with WorkerLike {
    def doWork() Int = 7
}

class Other with WorkerLike {
    def doWork() Int = 3
}

def main() Int {
    some MaybeInt = someInt(5)
    none MaybeInt = noneInt()
    printOption(some)
    printOption(none)
    Term.println(matchSingleLine(some))
    Term.println(matchSingleLine(none))

    exactPair = match (1, 2) {
        (1, 2) => "exact tuple"
        _ => "miss"
    }

    Term.println(exactPair)

    exactPair2 = match (2, 3) {
        (a Int, b Int) => "exact tuple " + a + "-" + b
        _ => "miss"
    }
    Term.println(exactPair2)

    exactPair3 = match (4, 5) {
        (a, b Int) => "exact tuple " + a + "-" + b
        _ => "miss"
    }
    Term.println(exactPair3)

    exactValue = match 5 {
        4 => "nope"
        5 => "exact value"
        _ => "miss"
    }
    Term.println(exactValue)

    workerLike WorkerLike = Worker()
    Term.println(match workerLike {
        worker Worker => worker.doWork()
        _ Other => 100
        _ => 0
    })

    pair = (1, 2)
    match pair {
        (left, right) => {
            Term.println("tuple " + left + " " + right)
        }
    }

    0
}
