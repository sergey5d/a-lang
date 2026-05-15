package alang.stdlib;

import java.util.HashMap;
import java.util.Iterator;
import java.util.function.BiFunction;
import java.util.function.Predicate;

public final class Map<K, V> implements Iterable<Tuple2<K, V>> {
    @FunctionalInterface
    public interface Mapper2<A, B, R> {
        R apply(A a, B b);
    }

    @FunctionalInterface
    public interface Mapper4<A, B, C, D, R> {
        R apply(A a, B b, C c, D d);
    }

    private final HashMap<K, V> items;

    private Map(HashMap<K, V> items) {
        this.items = items;
    }

    public Map() {
        this(new HashMap<>());
    }

    public static <K, V> Map<K, V> of() {
        return new Map<>();
    }

    public Map<K, V> put(K key, V value) {
        this.items.put(key, value);
        return this;
    }

    public <X> List<X> map(Mapper2<? super K, ? super V, ? extends X> f) {
        List<X> out = new List<>();
        for (java.util.Map.Entry<K, V> entry : this.items.entrySet()) {
            out.append(f.apply(entry.getKey(), entry.getValue()));
        }
        return out;
    }

    public <X> Map<K, X> mapValues(java.util.function.Function<? super V, ? extends X> f) {
        HashMap<K, X> out = new HashMap<>();
        for (java.util.Map.Entry<K, V> entry : this.items.entrySet()) {
            out.put(entry.getKey(), f.apply(entry.getValue()));
        }
        return new Map<>(out);
    }

    public <X> List<X> flatMap(Mapper2<? super K, ? super V, ? extends List<X>> f) {
        List<X> out = new List<>();
        for (java.util.Map.Entry<K, V> entry : this.items.entrySet()) {
            List<X> mapped = f.apply(entry.getKey(), entry.getValue());
            for (X value : mapped) {
                out.append(value);
            }
        }
        return out;
    }

    public Map<K, V> filter(Predicate<Tuple2<K, V>> f) {
        HashMap<K, V> out = new HashMap<>();
        for (java.util.Map.Entry<K, V> entry : this.items.entrySet()) {
            Tuple2<K, V> tuple = new Tuple2<>(entry.getKey(), entry.getValue());
            if (f.test(tuple)) {
                out.put(entry.getKey(), entry.getValue());
            }
        }
        return new Map<>(out);
    }

    public <X> X fold(X initial, Mapper3<X, K, V, X> f) {
        X result = initial;
        for (java.util.Map.Entry<K, V> entry : this.items.entrySet()) {
            result = f.apply(result, entry.getKey(), entry.getValue());
        }
        return result;
    }

    @FunctionalInterface
    public interface Mapper3<A, B, C, R> {
        R apply(A a, B b, C c);
    }

    public Option<Tuple2<K, V>> reduce(Mapper4<? super K, ? super V, ? super K, ? super V, Tuple2<K, V>> f) {
        Iterator<java.util.Map.Entry<K, V>> it = this.items.entrySet().iterator();
        if (!it.hasNext()) {
            return Option.none();
        }
        java.util.Map.Entry<K, V> first = it.next();
        K currentKey = first.getKey();
        V currentValue = first.getValue();
        while (it.hasNext()) {
            java.util.Map.Entry<K, V> next = it.next();
            Tuple2<K, V> result = f.apply(currentKey, currentValue, next.getKey(), next.getValue());
            currentKey = result._1;
            currentValue = result._2;
        }
        return Option.some(new Tuple2<>(currentKey, currentValue));
    }

    public boolean exists(Predicate<Tuple2<K, V>> f) {
        for (java.util.Map.Entry<K, V> entry : this.items.entrySet()) {
            if (f.test(new Tuple2<>(entry.getKey(), entry.getValue()))) {
                return true;
            }
        }
        return false;
    }

    public boolean forAll(Predicate<Tuple2<K, V>> f) {
        for (java.util.Map.Entry<K, V> entry : this.items.entrySet()) {
            if (!f.test(new Tuple2<>(entry.getKey(), entry.getValue()))) {
                return false;
            }
        }
        return true;
    }

    public void forEach(Mapper2<? super K, ? super V, ?> f) {
        for (java.util.Map.Entry<K, V> entry : this.items.entrySet()) {
            f.apply(entry.getKey(), entry.getValue());
        }
    }

    public Option<V> get(K key) {
        if (!this.items.containsKey(key)) {
            return Option.none();
        }
        return Option.some(this.items.get(key));
    }

    public boolean contains(K key) {
        return this.items.containsKey(key);
    }

    public long size() {
        return this.items.size();
    }

    @Override
    public Iterator<Tuple2<K, V>> iterator() {
        Iterator<java.util.Map.Entry<K, V>> base = this.items.entrySet().iterator();
        return new Iterator<Tuple2<K, V>>() {
            @Override
            public boolean hasNext() {
                return base.hasNext();
            }

            @Override
            public Tuple2<K, V> next() {
                java.util.Map.Entry<K, V> entry = base.next();
                return new Tuple2<>(entry.getKey(), entry.getValue());
            }
        };
    }
}
