interface List[T] with Iterable[T] {

    def append(value T) List[T]

    def get(index Int) Option[T]

    def head() Option[T]

    def tail() List[T]

    def remove(index Int) Option[T]

    def size() Int

    def forEach(f T -> Unit) Unit
}
