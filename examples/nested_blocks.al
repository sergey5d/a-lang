# EXPECT:
# xxx
# 8
# 6

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

    Term.println(a1)
    Term.println(v)
}
