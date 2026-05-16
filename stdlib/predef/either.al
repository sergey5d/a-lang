enum Either[L, R] {
    def isLeft() Bool = match this {
        Left(_) => true
        Right(_) => false
    }

    def isRight() Bool = !this.isLeft()

    def isFailure() Bool = this.isLeft()

    def unwrap() R = match this {
        Left(_) => OS.panic("Either has no right value")
        Right(value) => value
    }

    def getLeft() L = match this {
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

    case Left {
        value L
    }

    case Right {
        value R
    }
}
