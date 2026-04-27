# EXPECT:
# xxx
# 8
# 6
# 20
# 1
# 20
# 42

enum MaybeInt {
    case NoneX
    case SomeX {
        value Int
    }
}

def main() Unit {
    a1 = {
        1 + 7
    }
    {
        Term.println("xxx")
    }
    v := {
        a = 5
        {
            a + 1
        }
    }
    branch := {
        {
            if false {
                10
            } else {
                20
            }
        }
    }
    yielded = {
        for item <- [1, 2, 3] yield {
            if item % 2 == 0 {
                item * 10
            } else {
                item
            }
        }
    }
    matched = {
        match MaybeInt.SomeX(42) {
            SomeX(value) => value
            MaybeInt.NoneX => 0
        }
    }

    Term.println(a1)
    Term.println(v)
    Term.println(branch)
    Term.println(yielded.get(0).getOr(0))
    Term.println(yielded.get(1).getOr(0))
    Term.println(matched)
}
