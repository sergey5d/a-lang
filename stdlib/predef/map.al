interface Map[K, V] with Iterable[(K, V)] {
    def set(key K, value V) Map[K, V]
    def iterator() Iterator[(K, V)]
    def map[X](f (K, V) -> X) List[X]
    def flatMap[X](f (K, V) -> List[X]) List[X]
    def forEach(f (K, V) -> Unit)
    def get(key K) Option[V]
    def contains(key K) Bool
    def size() Int
}
