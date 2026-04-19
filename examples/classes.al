# EXPECT:
# a fields 15 Sammy
# a alt 6 5
# b manual 5 Bobby
# combo 11

class A {
    age Int
    name String

    private malnutritioned Bool := false

    def this(maturity Int) {
        this(age = maturity - 15, name = "5")
    }

    def trueAge() Int = age

    def trueName() String = name
}

class B {
    private age Int
    private name String
    private malnutritioned Bool := false
    private unassigned Float := ?

    def this(age Int, name String) {
        this.age = age
        this.name = name
        unassigned := 1.1
    }

    def trueAge() Int = age

    def trueName() String = name
}

def main() Unit {
    aFromFields A = A(age = 15, name = "Sammy")
    aFromAlt A = A(maturity = 21)
    bManual B = B(5, "Bobby")

    combo Int = aFromAlt.trueAge() + bManual.trueAge()

    if aFromFields.trueName() == "Sammy" {
        Term.println("a fields", aFromFields.trueAge(), aFromFields.trueName())
        Term.println("a alt", aFromAlt.trueAge(), aFromAlt.trueName())
        Term.println("b manual", bManual.trueAge(), bManual.trueName())
        Term.println("combo", combo)
    }
}
