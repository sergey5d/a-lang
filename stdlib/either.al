class Either[L, R] with Unwrappable[R] {
    private var rightSet Bool
    private var left L
    private var right R
}

impl Either[L, R] {
    def isLeft() Bool = !rightSet
    def isRight() Bool = rightSet
    def isFailure() Bool = !rightSet
    def unwrap() R = right
    def getLeft() L = left
    def getOr(defaultValue R) R =
        if rightSet {
            right
        } else {
            defaultValue
        }
    def map[X](f R -> X) Either[L, X] =
        if rightSet {
            Right(f(right))
        } else {
            Left(left)
        }
}
