class Result[T, E] {
    hidden var ok Bool
    hidden var value T
    hidden var error E
}

impl Result[T, E] {
    def isOk() Bool = ok
    def isErr() Bool = !ok
    def isFailure() Bool = !ok
    def unwrap() T = value
    def getError() E = error
    def getOr(defaultValue T) T =
        if ok then value else defaultValue

    def map[X](f T -> X) Result[X, E] =
        if ok then Ok(f(value)) else Err(error)
}
