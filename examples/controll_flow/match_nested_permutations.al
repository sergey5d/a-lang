# EXPECT:
# class in class 11
# record in class 3 usd
# class in record 7
# tuple in tuple 1 2 3
# tuple tuple tuple 20 21 22 23
# tuple in class 4 5
# class in tuple 6 9
# record in tuple 8 eur
# tuple in record 10 11
# enum in enum on
# class in enum 12
# record in enum 13 cad
# tuple in enum 14 15
# enum in class off
# enum in record on
# enum in tuple off 16

class Apple {
    size Int
}

class AppleBox {
    apple Apple
}

record Amount {
    count Int
    label Str
}

class AmountBox {
    amount Amount
}

record AppleRecord {
    apple Apple
}

class PairHolder {
    pair (Int, Int)
}

record PairRecord {
    pair (Int, Int)
}

enum InnerFlag {
    case Off
    case On
}

enum OuterFlag {
    case Empty
    case Wrap {
        value InnerFlag
    }
}

enum MaybeApple {
    case NoneX
    case SomeX {
        value Apple
    }
}

enum MaybeAmount {
    case NoneX
    case SomeX {
        value Amount
    }
}

enum MaybePair {
    case NoneX
    case SomeX {
        value (Int, Int)
    }
}

class FlagBox {
    value InnerFlag
}

record FlagRecord {
    value InnerFlag
}

def main() Unit {
    classInClass = match AppleBox(Apple(11)) {
        AppleBox(Apple(size)) => "class in class " + size
    }
    OS.println(classInClass)

    recordInClass = match AmountBox(Amount(3, "usd")) {
        AmountBox(Amount(count, label)) => "record in class " + count + " " + label
    }
    OS.println(recordInClass)

    classInRecord = match AppleRecord(Apple(7)) {
        AppleRecord(Apple(size)) => "class in record " + size
    }
    OS.println(classInRecord)

    tupleInTuple = match ((1, 2), 3) {
        ((left, right), tail) => "tuple in tuple " + left + " " + right + " " + tail
    }
    OS.println(tupleInTuple)

    tupleTupleTuple = match ((20, 21), (22, 23)) {
        ((left1, right1), (left2, right2)) => "tuple tuple tuple " + left1 + " " + right1 + " " + left2 + " " + right2
    }
    OS.println(tupleTupleTuple)

    tupleInClass = match PairHolder((4, 5)) {
        PairHolder((left, right)) => "tuple in class " + left + " " + right
    }
    OS.println(tupleInClass)

    classInTuple = match (Apple(6), 9) {
        (Apple(size), tail) => "class in tuple " + size + " " + tail
    }
    OS.println(classInTuple)

    recordInTuple = match (Amount(8, "eur"), 0) {
        (Amount(count, label), _) => "record in tuple " + count + " " + label
    }
    OS.println(recordInTuple)

    tupleInRecord = match PairRecord((10, 11)) {
        PairRecord((left, right)) => "tuple in record " + left + " " + right
    }
    OS.println(tupleInRecord)

    enumInEnum = match OuterFlag.Wrap(InnerFlag.On) {
        Wrap(InnerFlag.On) => "enum in enum on"
        Wrap(InnerFlag.Off) => "enum in enum off"
        OuterFlag.Empty => "enum in enum empty"
    }
    OS.println(enumInEnum)

    classInEnum = match MaybeApple.SomeX(Apple(12)) {
        SomeX(Apple(size)) => "class in enum " + size
        MaybeApple.NoneX => "class in enum none"
    }
    OS.println(classInEnum)

    recordInEnum = match MaybeAmount.SomeX(Amount(13, "cad")) {
        SomeX(Amount(count, label)) => "record in enum " + count + " " + label
        MaybeAmount.NoneX => "record in enum none"
    }
    OS.println(recordInEnum)

    tupleInEnum = match MaybePair.SomeX((14, 15)) {
        SomeX((left, right)) => "tuple in enum " + left + " " + right
        MaybePair.NoneX => "tuple in enum none"
    }
    OS.println(tupleInEnum)

    enumInClass = match FlagBox(InnerFlag.Off) {
        FlagBox(InnerFlag.On) => "enum in class on"
        FlagBox(InnerFlag.Off) => "enum in class off"
    }
    OS.println(enumInClass)

    enumInRecord = match FlagRecord(InnerFlag.On) {
        FlagRecord(InnerFlag.On) => "enum in record on"
        FlagRecord(InnerFlag.Off) => "enum in record off"
    }
    OS.println(enumInRecord)

    enumInTuple = match (InnerFlag.Off, 16) {
        (InnerFlag.On, value) => "enum in tuple on " + value
        (InnerFlag.Off, value) => "enum in tuple off " + value
    }
    OS.println(enumInTuple)
}
