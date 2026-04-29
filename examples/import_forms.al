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

    OS.println(named.label())
    OS.println(aliasA.label())
    OS.println(valueB.label())
    OS.println(C(4) + things.C(5))
    return 0
}
