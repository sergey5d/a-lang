# EXPECT:
# a fields 15 Sammy
# a alt 6 5
# b manual 5 Bobby
# combo 11

class A {
    age Int
    name Str

    private malnutritioned Bool := false
}

impl A {
    # explicit call to primary constructor
    def this(maturity Int) = this(maturity - 15, "5")

    def this(name Str) {
        # no need to call primary constructor if we init all fields in secondary constructor
        this.name = name
        age = 15
    }

    def trueAge() Int = age

    def trueName() Str = name
}

class B {
    # immutable member variables
    private age Int
    private name Str
    # initialized mutable variable
    private malnutritioned Bool := false
    # not yet initialized mutable variable
    private unassigned Float := ?
    # not yet initialized immutable variable (syntax is for consistency only)
    private questionable Int = ?
}

impl B {
    def this(age Int, name Str) {
        this.age = age
        this.name = name
        this.unassigned := 1.1
        this.questionable = 1
    }

    def trueAge() Int = age

    def trueName() Str = name
}

def main() Unit {
    aFromFields A = A(age = 15, name = "Sammy")
    aFromAlt A = A(maturity = 21)
    bManual B = B(5, "Bobby")

    combo Int = aFromAlt.trueAge() + bManual.trueAge()

    if aFromFields.trueName() == "Sammy" {
        OS.println("a fields", aFromFields.trueAge(), aFromFields.trueName())
        OS.println("a alt", aFromAlt.trueAge(), aFromAlt.trueName())
        OS.println("b manual", bManual.trueAge(), bManual.trueName())
        OS.println("combo", combo)
    }
}
