# EXPECT:
# tuple 5 hehe
# tuple inferred 6 there
# tuple mixed 8 mixed
# tuple mixed2 9 mixed2
# record 7 world
# record inferred 10 infer
# record mixed 11 partial
# class 9 boxed
# class inferred 12 class
# class mixed 13 typed
# 0

record Pair {
    left Int
    right String
}

class Box {
    value Int
    label String
}

def main() Int {
    a Int, b String = (5, "hehe")
    Term.println("tuple", a, b)

    inferredTupleLeft, inferredTupleRight = (6, "there")
    Term.println("tuple inferred", inferredTupleLeft, inferredTupleRight)

    mixedTupleLeft Int, mixedTupleRight = (8, "mixed")
    Term.println("tuple mixed", mixedTupleLeft, mixedTupleRight)

    mixedTuple2Left, mixedTuple2Right String = (9, "mixed2")
    Term.println("tuple mixed2", mixedTuple2Left, mixedTuple2Right)

    c Int, d String = Pair(7, "world")
    Term.println("record", c, d)

    inferredRecordLeft, inferredRecordRight = Pair(10, "infer")
    Term.println("record inferred", inferredRecordLeft, inferredRecordRight)

    mixedRecordLeft Int, mixedRecordRight = Pair(11, "partial")
    Term.println("record mixed", mixedRecordLeft, mixedRecordRight)

    e Int, f String = Box(9, "boxed")
    Term.println("class", e, f)

    inferredClassLeft, inferredClassRight = Box(12, "class")
    Term.println("class inferred", inferredClassLeft, inferredClassRight)

    mixedClassLeft Int, mixedClassRight = Box(13, "typed")
    Term.println("class mixed", mixedClassLeft, mixedClassRight)

    0
}
