# EXPECT:
# some 6
# none missing
# pair 5-no-right

def plusOne(value Option[Int]) Result[Int, Str] {
    guard item <- value: Err("missing")
    Ok(item + 1)
}

def pairwise(left Option[Int], right Option[Str]) { count Int, label Str } = {
    guard count <- left {
        record { count = 0, label = "no-left" }
    }
    guard label <- right {
        record { count = count, label = "no-right" }
    }
    record(count, label)
}

def main() Unit {
    some = plusOne(Some(5))
    none = plusOne(None())
    pair = pairwise(Some(5), None())

    OS.println("some ${some.getOr(0)}")
    OS.println("none ${none.getError()}")
    OS.println("pair ${pair.count}-${pair.label}")
}
