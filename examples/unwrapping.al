# EXPECT:
# option some 6
# option none true
# result ok 7
# result err bad
# either right 8
# either left nope
# either combo 11
# either combo left size bad

def plusOneOption(value Option[Int]) Option[Int] {
    item <- value
    return Some(item + 1)
}

def plusOneResult(value Result[Int, Str]) Result[Int, Str] {
    item <- value
    return Ok(item + 1)
}

def plusOneEither(value Either[Str, Int]) Either[Str, Int] {
    item <- value
    return Right(item + 1)
}

def twoEithers(value Either[Str, Int], value2 Either[Str, Str]) Either[Str, Int] {
    item <- value
    str <- value2
    size <- value2.map((s Str) -> s.size())
    return Right(item + str.size() / 2 + size)
}

def main() {
    optionSome = plusOneOption(Some(5))
    optionNone = plusOneOption(None())

    Term.println("option some ${optionSome.getOr(0)}")
    Term.println("option none ${optionNone.isEmpty()}")

    resultOk = plusOneResult(Ok(6))
    resultErr = plusOneResult(Err("bad"))

    Term.println("result ok ${resultOk.getOr(0)}")
    Term.println("result err ${resultErr.getError()}")

    eitherRight = plusOneEither(Right(7))
    eitherLeft = plusOneEither(Left("nope"))
    Term.println("either right ${eitherRight.getOr(0)}")
    Term.println("either left ${eitherLeft.getLeft()}")

    if true {
        combo = twoEithers(Right(7), Right("abc"))
        comboLeft = twoEithers(Right(7), Left("size bad"))
        Term.println("either combo ${combo.getOr(0)}")
        Term.println("either combo left ${comboLeft.getLeft()}")
    }
}
