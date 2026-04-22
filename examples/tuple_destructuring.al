# EXPECT:
# tuple 5 hehe
# tuple inferred 6 there
# tuple mixed 8 mixed
# tuple mixed2 9 mixed2
# tuple pair 20 pair
# tuple skip only unused
# tuple skip 14 xxx
# 0

def main() Int {
    a Int, b String = (5, "hehe")
    Term.println("tuple", a, b)

    inferredTupleLeft, inferredTupleRight = (6, "there")
    Term.println("tuple inferred", inferredTupleLeft, inferredTupleRight)

    mixedTupleLeft Int, mixedTupleRight = (8, "mixed")
    Term.println("tuple mixed", mixedTupleLeft, mixedTupleRight)

    mixedTuple2Left, mixedTuple2Right String = (9, "mixed2")
    Term.println("tuple mixed2", mixedTuple2Left, mixedTuple2Right)

    tuplePairLeft, tuplePairRight = (20, "pair")
    Term.println("tuple pair", tuplePairLeft, tuplePairRight)

    _, skippedOnlyTupleValue = (21, "unused")
    Term.println("tuple skip only", skippedOnlyTupleValue)

    skippedTupleLeft Int, _, skippedTupleRight String = (14, "drop", "xxx")
    Term.println("tuple skip", skippedTupleLeft, skippedTupleRight)

    0
}
