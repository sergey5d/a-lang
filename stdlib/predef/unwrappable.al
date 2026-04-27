interface Unwrappable[T] {
    def isFailure() Bool
    def unwrap() T
}
