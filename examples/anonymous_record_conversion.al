# EXPECT:
# record Ada 10
# public Ben 12 NYC
# vars 3 4
# ctor Mia 99
# mixed Liam 8 5

record User {
    name Str
    age Int
}

class Person {
    name Str
    age Int
    city Str = "NYC"
}

class MutablePoint {
    var x Int
    var y Int
}

class SecretBadge {
    name Str
    private code Int
}

impl SecretBadge {
    def init(name Str) {
        this.name = name
        this.code = 99
    }

    def codeValue() Int = code
}

class MixedProfile {
    name Str
    age Int
    private score Int = 5
}

impl MixedProfile {
    def scoreValue() Int = score
}

def main() Unit {
    userRecord = record {
        name = "Ada"
        age = 10
    }
    user User = User(userRecord)

    person Person = Person(record("Ben", 12))

    pointRecord = record {
        x = 3
        y = 4
    }
    point MutablePoint = MutablePoint(pointRecord)

    badge SecretBadge = SecretBadge(record {
        name = "Mia"
    })

    profile MixedProfile = MixedProfile(record {
        name = "Liam"
        age = 8
    })

    OS.println("record", user.name, user.age)
    OS.println("public", person.name, person.age, person.city)
    OS.println("vars", point.x, point.y)
    OS.println("ctor", badge.name, badge.codeValue())
    OS.println("mixed", profile.name, profile.age, profile.scoreValue())
}
