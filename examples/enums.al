# EXPECT:
# reddish: false
# some defined: true
# none: true

# Enums might have 2 flavors.
# Fully defined where all attributes have values or partially defined see Option.Some

enum Color {
    color String
    temperature Int

    def isReddish() Bool = temperature % 5 == 0

    case Black {
        color = "xxx"
        temperature = 1
    }
    case Red {
        color = "xxx2"
        temperature = 2
    }
}

enum Option[T] {

    def isDefined() Bool = this != None

    case None
    case Some {
        value T
    }
}

black = Color.Black
reddish = black.isReddish()

someInt = Option.Some(5)
noneInt = Option.None

someDefined = someInt.isDefined()

def main() Unit {
    Term.println("reddish:", reddish)
    Term.println("some defined:", someDefined)
    Term.println("none:", noneInt == Option.None)
}
