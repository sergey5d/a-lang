# EXPECT:
# tuple 5 hehe
# tuple inferred 6 there
# tuple mixed 8 mixed
# tuple mixed2 9 mixed2
# tuple skip 14 xxx
# record 7 world
# record inferred 10 infer
# record mixed 11 partial
# record skip 15 kept
# class 9 boxed
# class inferred 12 class
# class mixed 13 typed
# class skip 16 class-skip
# 0

record Pair {
    left Int
    right String
}

record Triple {
    first Int
    middle String
    last String
}

class Box {
    value Int
    label String
}

class Crate {
    value Int
    hidden String
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

    skippedTupleLeft Int, _, skippedTupleRight String = (14, "drop", "xxx")
    Term.println("tuple skip", skippedTupleLeft, skippedTupleRight)

    c Int, d String = Pair(7, "world")
    Term.println("record", c, d)

    inferredRecordLeft, inferredRecordRight = Pair(10, "infer")
    Term.println("record inferred", inferredRecordLeft, inferredRecordRight)

    mixedRecordLeft Int, mixedRecordRight = Pair(11, "partial")
    Term.println("record mixed", mixedRecordLeft, mixedRecordRight)

    skippedRecordLeft Int, _, skippedRecordRight String = Triple(15, "drop", "kept")
    Term.println("record skip", skippedRecordLeft, skippedRecordRight)

    e Int, f String = Box(9, "boxed")
    Term.println("class", e, f)

    inferredClassLeft, inferredClassRight = Box(12, "class")
    Term.println("class inferred", inferredClassLeft, inferredClassRight)

    mixedClassLeft Int, mixedClassRight = Box(13, "typed")
    Term.println("class mixed", mixedClassLeft, mixedClassRight)

    skippedClassLeft Int, _, skippedClassRight String = Crate(16, "drop", "class-skip")
    Term.println("class skip", skippedClassLeft, skippedClassRight)

    0
}
