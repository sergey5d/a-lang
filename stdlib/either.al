enum Either[L, R] {

    def isLeft() Bool
    def isRight() Bool

    def map[T](f R -> T) Either[L, T]
    def mapLeft[T](f L -> T) Either[T, R]

    def toResult() Result[L, R]
    def toOption() Option[R]
    def toList() List[R]

    def swap() Either[R, L]

    def contains(v R) Bool

    def exists(f R -> Bool) Bool
    def forAll(f R -> Bool) Bool

    def fold[C](fl L -> C, fr R -> C) C

    def forEach(f R -> Unit)

    def getOr(f () -> R) R

    def orElse(f () -> Either[L, R]) Either[L, R]

    case Left[L] {

        private val L

        def this(val L) {
            this.val = val
        }

        def isLeft() Bool = true
        def isRight() Bool = false

        def map[T](f R -> T) Either[L, T] = this
        def mapLeft[T](f L -> T) Either[T, R] = this

        def toResult() Result[L, R] = Err(val)
        def toOption() Option[R] = None()
        def toList() List[R] = []

        def swap() Either[R, L] = Right(val)

        def contains(v R) Bool = false
        def exists(f R -> Bool) Bool = false
    
        def forAll(f R -> Bool) Bool = false

        def fold[T](fl L -> T, fr R -> T) T = fl(val)

        def forEach(f R -> Unit) = ()

        def getOr(f () -> R) R = f()

        def orElse(f () -> Either[L, R]) Either[L, R] = f()
    }

    case Right[R] {

        private val R

        def this(val R) {
            this.val = val
        }

        def isLeft() Bool = false
        def isRight() Bool = true

        def map[T](f R -> T) Either[L, T] = Right(f(val))
        def mapLeft[T](f L -> T) Either[T, R] = this

        def toResult() Result[L, R] = Ok(val)
        def toOption() Option[R] = Some(val)
        def toList() List[R] = [val]

        def swap() Either[R, L] = Left(val)

        def contains(v R) Bool = v == val
        def exists(f R -> Bool) Bool = f(val)
    
        def forAll(f R -> Bool) Bool = f(val)

        def fold[T](fl L -> T, fr R -> T) T = fr(val)

        def forEach(f R -> Unit) = f(val)

        def getOr(f () -> R) R = val

        def orElse(f () -> Either[L, R]) Either[L, R] = this
    }
}
