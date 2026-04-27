class Option[T] with Unwrappable[T] {
    private set Bool := ?
    private value T := ?

    def this() {
        this.set = false
    }

    def this(value T) {
        this.set = true
        this.value := value
    }

    def isSet() Bool = set

    def isEmpty() Bool = !set

    impl def isFailure() Bool = !set

    def get() T = value

    impl def unwrap() T = value

    def getOr(defaultValue T) T =
        if set {
            value
        } else {
            defaultValue
        }

    def map[X](f T -> X) Option[X] =
        if set {
            Some(f(value))
        } else {
            None()
        }
}
