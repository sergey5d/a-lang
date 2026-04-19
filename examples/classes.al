# IGNORE
# EXPECT:
# combo 11

class A {
    age Int
    name String

    private malnutritioned := false

    constructor(maturity Int) {
        constructor(age = maturity - 15, name = "5")
    }

    def trueAge() Int = age

    def trueName() String = name
}

aInstance = A(
    age = 15,
    name = "Sammy"
)

aInstance2 = A(
    maturity = 25
 )