enum Result[T, E] {
    def isOk() Bool = match this {
        Ok(_) => true
        Err(_) => false
    }

    def isErr() Bool = !this.isOk()

    def expect() T = match this {
        Ok(value) => value
        Err(_) => OS.panic("Result has no success value")
    }

    def getError() E = match this {
        Ok(_) => OS.panic("Result has no error value")
        Err(error) => error
    }

    def getOr(defaultValue T) T = match this {
        Ok(value) => value
        Err(_) => defaultValue
    }

    def map[X](f T -> X) Result[X, E] = match this {
        Ok(value) => Ok(f(value))
        Err(error) => Err(error)
    }

    case Ok {
        value T
    }

    case Err {
        error E
    }
}
