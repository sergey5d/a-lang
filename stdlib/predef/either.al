enum Either[L, R] {

    case Left {
        value L
    }

    case Right {
        value R
    }

    def isLeft() Bool = match this {
        Left(_) => true
        Right(_) => false
    }

    def isRight() Bool = !this.isLeft()

    def expectRight() R = match this {
        Left(_) => OS.panic("Either has no right value")
        Right(value) => value
    }

    def expectLeft() L = match this {
        Left(value) => value
        Right(_) => OS.panic("Either has no left value")
    }

    def getOr(defaultValue R) R = match this {
        Left(_) => defaultValue
        Right(value) => value
    }

    def map[X](f R -> X) Either[L, X] = match this {
        Left(value) => Left(value)
        Right(value) => Right(f(value))
    }

    def mapLeft[X](f L -> X) Either[X, R] = match this {
        Left(value) => Left(f(value))
        Right(value) => Right(value)
    }

    def flatMap[X](f R -> Either[L, X]) Either[L, X] = match this {
        Left(value) => Left(value)
        Right(value) => f(value)
    }

    def toOption() Option[R] = match this {
        Left(_) => None()
        Right(value) => Some(value)
    }

    def toResult() Result[R, L] = match this {
        Left(value) => Err(value)
        Right(value) => Ok(value)
    }
}
