class Either[L, R] with Unwrappable[R] {
    private rightSet Bool := ?
    private left L := ?
    private right R := ?

    def isLeft() Bool = !rightSet
    def isRight() Bool = rightSet
    impl def isFailure() Bool = !rightSet
    impl def unwrap() R = right
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
