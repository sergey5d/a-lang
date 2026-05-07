class Result[T, E] with Unwrappable[T] {
    private var ok Bool
    private var value T
    private var error E
}

impl Result[T, E] {
    def isOk() Bool = ok

    def isErr() Bool = !ok

    def isFailure() Bool = !ok

    def unwrap() T = value

    def getError() E = error

    def getOr(defaultValue T) T =
        if ok {
            value
        } else {
            defaultValue
        }

    def map[X](f T -> X) Result[X, E] =
        if ok {
            Ok(f(value))
        } else {
            Err(error)
        }
}
