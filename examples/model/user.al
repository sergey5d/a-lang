# EXPECT:
# Ada
# 36

package user

class Person {
    name String
    age Int

    def name() String = name

    def age() Int = age
}

def main() Unit {
    person = Person(name = "Ada", age = 36)
    Term.println(person.name())
    Term.println(person.age())
}
