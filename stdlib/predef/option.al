enum Option[T] {
    def isSet() Bool = match this {
        Some(_) => true
        Option.None => false
    }

    def isEmpty() Bool = !this.isSet()

    def expect() T = match this {
        Some(value) => value
        Option.None => OS.panic("Option has no value")
    }

    def getOr(defaultValue T) T = match this {
        Some(value) => value
        Option.None => defaultValue
    }

    def getOrElse(defaultValue T) T = this.getOr(defaultValue)

    def map[X](f T -> X) Option[X] = match this {
        Some(value) => Some(f(value))
        Option.None => None()
    }

    case None
    case Some {
        value T
    }
}
