# EXPECT:
# A
# A
# B
# 11
# 0

package app

import model/things
import model/things/A
import model/things/A as AliasA
import model/things/{B as AliasB, Named}
import model/things/*

def main() Int {
    value A = A()
    aliasA AliasA = AliasA()
    valueB AliasB = AliasB()
    named Named = value

    Term.println(named.label())
    Term.println(aliasA.label())
    Term.println(valueB.label())
    Term.println(C(4) + things.C(5))
    return 0
}
