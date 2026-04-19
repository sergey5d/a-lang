# EXPECT:
# hello, Ada
# 36

package app
import util
import model/user

def main() Unit {
    person = user.Person(name = "Ada", age = 36)
    Term.println(util.greet(person.name()))
    Term.println(person.age())
}
