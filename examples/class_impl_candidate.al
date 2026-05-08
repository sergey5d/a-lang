# SKIP
#
# Candidate class shape where storage stays in `class` and behavior moves into
# a separate `impl` block.

class Person with Named, Aged {

    firstName Str
    lastName Str
    age Int
    city Str

    private archived Bool : = false
    private var internalScore Int = 1

    #private {
    #    archived2 Bool
    #    internalScore2 Int
    #}
}

impl Person {

    #replace this to self?

    def init(firstName Str, lastName Str) {
        this.firstName = firstName
        this.lastName = lastName
        this.age = 18
        this.city = "unknown"
    }

    private def calc() Int {
        firstName.size() + lastName.size()
    }

    def fullName() Str = firstName + " " + lastName

    def isAdult() Bool = age >= 18

    def moveTo(newCity Str) Unit {
        city := newCity
    }

    def celebrateBirthday() Unit {
        age += 1
        internalScore += 10
    }

    def archive() Unit {
        archived := true
    }

    def debugLabel() Str =
        fullName() + "@" + city + ":" + age + ":" + internalScore + ":" + archived
}
