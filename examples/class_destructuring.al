# EXPECT:
# class 9 boxed
# class inferred 12 class
# class mixed 13 typed
# class pair 24 class-pair
# class skip only class-only
# class skip 16 class-skip
# 0

class Box {
    value Int
    label Str
}

class Crate {
    value Int
    hidden Str
    label Str
}

def main() Int {
    e Int, f Str = Box(9, "boxed")
    Term.println("class", e, f)

    inferredClassLeft, inferredClassRight = Box(12, "class")
    Term.println("class inferred", inferredClassLeft, inferredClassRight)

    mixedClassLeft Int, mixedClassRight = Box(13, "typed")
    Term.println("class mixed", mixedClassLeft, mixedClassRight)

    classPairLeft, classPairRight = Box(24, "class-pair")
    Term.println("class pair", classPairLeft, classPairRight)

    _, skippedOnlyClassValue = Box(25, "class-only")
    Term.println("class skip only", skippedOnlyClassValue)

    skippedClassLeft Int, _, skippedClassRight Str = Crate(16, "drop", "class-skip")
    Term.println("class skip", skippedClassLeft, skippedClassRight)

    0
}
