interface List[T] with Iterable[T] {

    def append(value T) List[T]

    def map[X](f T -> X) List[X]

    def flatMap[X](f T -> List[X]) List[X]

    def sort(ordering Ordering[T]) List[T]

    def get(index Int) Option[T]

    def head() Option[T]

    def tail() List[T]

    def remove(index Int) Option[T]

    def size() Int

    def forEach(f T -> Unit) Unit
}
