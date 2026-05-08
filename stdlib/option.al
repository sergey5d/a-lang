class Option[T] with Unwrappable[T] {
    private var set Bool
    private var value T
}

impl Option[T] {
    def init() {
        this.set = false
    }

    def init(value T) {
        this.set = true
        this.value := value
    }

    def isSet() Bool = set

    def isEmpty() Bool = !set

    def isFailure() Bool = !set

    def get() T = value

    def unwrap() T = value

    def getOr(defaultValue T) T =
        if set {
            value
        } else {
            defaultValue
        }

    def getOrElse(defaultValue T) T = this.getOr(defaultValue)

    def map[X](f T -> X) Option[X] =
        if set {
            Some(f(value))
        } else {
            None()
        }
}
