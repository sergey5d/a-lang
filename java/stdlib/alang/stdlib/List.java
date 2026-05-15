package alang.stdlib;

import java.util.ArrayList;
import java.util.Comparator;
import java.util.Iterator;
import java.util.function.BiFunction;
import java.util.function.BinaryOperator;
import java.util.function.Consumer;
import java.util.function.Function;
import java.util.function.Predicate;

public final class List<T> implements Iterable<T> {
    private final ArrayList<T> items;

    private List(ArrayList<T> items) {
        this.items = items;
    }

    public List() {
        this(new ArrayList<>());
    }

    @SafeVarargs
    public static <T> List<T> of(T... values) {
        ArrayList<T> items = new ArrayList<>();
        for (T value : values) {
            items.add(value);
        }
        return new List<>(items);
    }

    public List<T> append(T value) {
        this.items.add(value);
        return this;
    }

    public <X> List<X> map(Function<? super T, ? extends X> f) {
        ArrayList<X> out = new ArrayList<>(this.items.size());
        for (T item : this.items) {
            out.add(f.apply(item));
        }
        return new List<>(out);
    }

    public <X> List<X> flatMap(Function<? super T, ? extends List<X>> f) {
        ArrayList<X> out = new ArrayList<>();
        for (T item : this.items) {
            List<X> mapped = f.apply(item);
            for (X value : mapped) {
                out.add(value);
            }
        }
        return new List<>(out);
    }

    public List<T> filter(Predicate<? super T> f) {
        ArrayList<T> out = new ArrayList<>();
        for (T item : this.items) {
            if (f.test(item)) {
                out.add(item);
            }
        }
        return new List<>(out);
    }

    public <X> X fold(X initial, BiFunction<? super X, ? super T, ? extends X> f) {
        X result = initial;
        for (T item : this.items) {
            result = f.apply(result, item);
        }
        return result;
    }

    public Option<T> reduce(BinaryOperator<T> f) {
        if (this.items.isEmpty()) {
            return Option.none();
        }
        T result = this.items.get(0);
        for (int i = 1; i < this.items.size(); i++) {
            result = f.apply(result, this.items.get(i));
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

    public List<T> sort(Comparator<? super T> ordering) {
        ArrayList<T> out = new ArrayList<>(this.items);
        out.sort(ordering);
        return new List<>(out);
    }

    public <X> List<Tuple2<T, X>> zip(List<X> other) {
        int size = Math.min(this.items.size(), other.items.size());
        ArrayList<Tuple2<T, X>> out = new ArrayList<>(size);
        for (int i = 0; i < size; i++) {
            out.add(new Tuple2<>(this.items.get(i), other.items.get(i)));
        }
        return new List<>(out);
    }

    public List<Tuple2<T, Long>> zipWithIndex() {
        ArrayList<Tuple2<T, Long>> out = new ArrayList<>(this.items.size());
        for (int i = 0; i < this.items.size(); i++) {
            out.add(new Tuple2<>(this.items.get(i), (long) i));
        }
        return new List<>(out);
    }

    public Option<T> get(long index) {
        int idx = (int) index;
        if (idx < 0 || idx >= this.items.size()) {
            return Option.none();
        }
        return Option.some(this.items.get(idx));
    }

    public Option<T> head() {
        if (this.items.isEmpty()) {
            return Option.none();
        }
        return Option.some(this.items.get(0));
    }

    public List<T> tail() {
        if (this.items.size() <= 1) {
            return new List<>();
        }
        ArrayList<T> out = new ArrayList<>(this.items.subList(1, this.items.size()));
        return new List<>(out);
    }

    public boolean isEmpty() {
        return this.items.isEmpty();
    }

    public Option<T> remove(long index) {
        int idx = (int) index;
        if (idx < 0 || idx >= this.items.size()) {
            return Option.none();
        }
        return Option.some(this.items.remove(idx));
    }

    public Option<T> removeLast() {
        if (this.items.isEmpty()) {
            return Option.none();
        }
        return Option.some(this.items.remove(this.items.size() - 1));
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
