interface Array[T] with Iterable[T] {

    def [](idx Int) T
    def [](idx Int, val T)

    def get(idx Int) Option[T]

    def first() Option[T]
    def last() Option[T]

    def zip[X](other Array[X]) Array[(T, X)]
    def zipWithIndex() Array[(T, Int)]
    def size() Int
}
