class Option[T] {
    private set Bool = ?
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
    def get() T = value
    def getOr(defaultValue T) T =
        if set {
            value
        } else {
            defaultValue
        }
}
