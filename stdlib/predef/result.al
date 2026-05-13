class Result[T, E] {
    priv var ok Bool
    priv var value T
    priv var error E
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
