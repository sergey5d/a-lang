# EXPECT:
# some 6
# none missing
# pair 5-no-right
# sum 9
# sumFail sum-missing

def plusOne(value Option[Int]) Result[Int, Str] {
    guard item <- value else: Err("missing")
    Ok(item + 1)
}

def pairwise(left Option[Int], right Option[Str]) { count Int, label Str } = {
    guard count <- left else {
        record { count = 0, label = "no-left" }
    }
    guard label <- right else {
        record { count = count, label = "no-right" }
    }
    record(count, label)
}

def sumBoth(left Option[Int], right Option[Int]) Result[Int, Str] {
    guard {
        a <- left
        b <- right
    } else {
        Err("sum-missing")
    }
    Ok(a + b)
}

def main() Unit {
    some = plusOne(Some(5))
    none = plusOne(None())
    pair = pairwise(Some(5), None())
    sum = sumBoth(Some(4), Some(5))
    sumFail = sumBoth(Some(4), None())

    OS.println("some ${some.getOr(0)}")
    OS.println("none ${none.getError()}")
    OS.println("pair ${pair.count}-${pair.label}")
    OS.println("sum ${sum.getOr(0)}")
    OS.println("sumFail ${sumFail.getError()}")
}
