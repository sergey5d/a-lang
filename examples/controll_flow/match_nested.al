# EXPECT:
# 1
# 2
# 9

enum BoolBox {
    case Wrap {
        value Bool
    }
    case Empty
}

enum PairBox {
    case Full {
        value (Int, Int)
    }
    case NoneX
}

def describeFlag(value BoolBox) Int =
    match value {
        Wrap(true) => 1
        Wrap(false) => 2
        BoolBox.Empty => 0
    }

def describePair(value PairBox) Int =
    match value {
        Full((left, right)) => left + right
        PairBox.NoneX => 0
    }

def main() Unit {
    OS.println(describeFlag(BoolBox.Wrap(true)))
    OS.println(describeFlag(BoolBox.Wrap(false)))
    OS.println(describePair(PairBox.Full((4, 5))))
}
