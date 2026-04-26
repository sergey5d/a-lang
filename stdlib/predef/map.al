interface Map[K, V] {
    def set(key K, value V) Map[K, V]
    def get(key K) Option[V]
    def contains(key K) Bool
    def size() Int
}
