interface Set[T] with Iterable[T] {
    def add(value T) Set[T]
    def iterator() Iterator[T]
    def map[X](f T -> X) Set[X]
    def flatMap[X](f T -> Set[X]) Set[X]
    def forEach(f T -> Unit)
    def contains(value T) Bool
    def size() Int
}
