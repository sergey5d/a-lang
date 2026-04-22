# EXPECT:
# record 7 world
# record inferred 10 infer
# record mixed 11 partial
# record pair 22 record-pair
# record skip only visible
# record skip 15 kept
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

def main() Int {
    c Int, d String = Pair(7, "world")
    Term.println("record", c, d)

    inferredRecordLeft, inferredRecordRight = Pair(10, "infer")
    Term.println("record inferred", inferredRecordLeft, inferredRecordRight)

    mixedRecordLeft Int, mixedRecordRight = Pair(11, "partial")
    Term.println("record mixed", mixedRecordLeft, mixedRecordRight)

    recordPairLeft, recordPairRight = Pair(22, "record-pair")
    Term.println("record pair", recordPairLeft, recordPairRight)

    _, skippedOnlyRecordValue = Pair(23, "visible")
    Term.println("record skip only", skippedOnlyRecordValue)

    skippedRecordLeft Int, _, skippedRecordRight String = Triple(15, "drop", "kept")
    Term.println("record skip", skippedRecordLeft, skippedRecordRight)

    0
}
