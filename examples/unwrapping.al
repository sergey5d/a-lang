# EXPECT:
# option some 6
# option none true
# result ok 7
# result err bad
# either right 8
# either left nope

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

def main() {
    optionSome = plusOneOption(Some(5))
    optionNone = plusOneOption(None())
    resultOk = plusOneResult(Ok(6))
    resultErr = plusOneResult(Err("bad"))
    eitherRight = plusOneEither(Right(7))
    eitherLeft = plusOneEither(Left("nope"))

    Term.println("option some ${optionSome.getOr(0)}")
    Term.println("option none ${optionNone.isEmpty()}")
    Term.println("result ok ${resultOk.getOr(0)}")
    Term.println("result err ${resultErr.getError()}")
    Term.println("either right ${eitherRight.getOr(0)}")
    Term.println("either left ${eitherLeft.getLeft()}")
}
