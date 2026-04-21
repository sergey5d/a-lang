interface List[T] with Iterable[T] {
    def append(value T) List[T]
    def get(index Int) Option[T]
    def size() Int
}
