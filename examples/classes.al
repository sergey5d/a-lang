# EXPECT:
# a fields 15 Sammy
# a alt 6 5
# b manual 5 Bobby
# combo 11

class A {
    age Int
    name String

    private malnutritioned Bool := false

    # explicit call to primary constructor
    def this(maturity Int) = this(maturity - 15, "5")

    def this(name String) {
        # no need to call primary constructor if we init all fields in secondary constructor
        this.name = name
        age = 15
    }

    def trueAge() Int = age

    def trueName() String = name
}

class B {
    # immutable member variables
    private age Int
    private name String
    # initialized mutable variable
    private malnutritioned Bool := false
    # not yet initialized mutable variable
    private unassigned Float := ?
    # not yet initialized immutable variable (syntax is for consistency only)
    private questionable Int = ?

    def this(age Int, name String) {
        this.age = age
        this.name = name
        this.unassigned := 1.1
        this.questionable = 1
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
