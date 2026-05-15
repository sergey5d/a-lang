package alang.stdlib;

import java.util.HashSet;
import java.util.Iterator;
import java.util.function.BiFunction;
import java.util.function.BinaryOperator;
import java.util.function.Consumer;
import java.util.function.Function;
import java.util.function.Predicate;

public final class Set<T> implements Iterable<T> {
    private final HashSet<T> items;

    private Set(HashSet<T> items) {
        this.items = items;
    }

    public Set() {
        this(new HashSet<>());
    }

    @SafeVarargs
    public static <T> Set<T> of(T... values) {
        HashSet<T> items = new HashSet<>();
        for (T value : values) {
            items.add(value);
        }
        return new Set<>(items);
    }

    public Set<T> add(T value) {
        this.items.add(value);
        return this;
    }

    public <X> Set<X> map(Function<? super T, ? extends X> f) {
        HashSet<X> out = new HashSet<>();
        for (T item : this.items) {
            out.add(f.apply(item));
        }
        return new Set<>(out);
    }

    public <X> Set<X> flatMap(Function<? super T, ? extends Set<X>> f) {
        HashSet<X> out = new HashSet<>();
        for (T item : this.items) {
            Set<X> mapped = f.apply(item);
            for (X value : mapped) {
                out.add(value);
            }
        }
        return new Set<>(out);
    }

    public Set<T> filter(Predicate<? super T> f) {
        HashSet<T> out = new HashSet<>();
        for (T item : this.items) {
            if (f.test(item)) {
                out.add(item);
            }
        }
        return new Set<>(out);
    }

    public <X> X fold(X initial, BiFunction<? super X, ? super T, ? extends X> f) {
        X result = initial;
        for (T item : this.items) {
            result = f.apply(result, item);
        }
        return result;
    }

    public Option<T> reduce(BinaryOperator<T> f) {
        Iterator<T> it = this.items.iterator();
        if (!it.hasNext()) {
            return Option.none();
        }
        T result = it.next();
        while (it.hasNext()) {
            result = f.apply(result, it.next());
        }
        return Option.some(result);
    }

    public boolean exists(Predicate<? super T> f) {
        for (T item : this.items) {
            if (f.test(item)) {
                return true;
            }
        }
        return false;
    }

    public boolean forAll(Predicate<? super T> f) {
        for (T item : this.items) {
            if (!f.test(item)) {
                return false;
            }
        }
        return true;
    }

    public boolean contains(T value) {
        return this.items.contains(value);
    }

    public long size() {
        return this.items.size();
    }

    @Override
    public void forEach(Consumer<? super T> consumer) {
        for (T item : this.items) {
            consumer.accept(item);
        }
    }

    @Override
    public Iterator<T> iterator() {
        return this.items.iterator();
    }
}
