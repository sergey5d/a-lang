# EXPECT:
# Ana is 10
# Ben is 12
# 1
# 20
# Ada-10
# 11
# Ana 10
# 6

def describe(user { name Str, age Int }) Str =
    user.name + " is " + user.age

def makeCounter(base Int) { count Int, next Int } =
    record {
        count = base
        next = base + 1
    }

def main() Unit {
    full = record {
        name = "Ana"
        age = 10
        city = "NYC"
    }
    smaller = record {
        name = "Ben"
        age = 12
    }
    a = 1
    inferred = record {
        count = a
    }
    mixed = record { a = 5, c = 7,
        b = 8
    }
    mixedShape = record { name = "Ada",
        age = 10
    }
    narrow { name Str, age Int } = full

    OS.println(describe(full))
    OS.println(describe(smaller))
    OS.println(inferred.count)
    OS.println(mixed.a + mixed.b + mixed.c)
    OS.println(mixedShape.name + "-" + mixedShape.age)
    counter = makeCounter(5)
    OS.println(counter.count + counter.next)
    OS.println(narrow.name + " " + narrow.age)
    typedCounter { count Int, next Int } = makeCounter(5)
    OS.println(typedCounter.next)
}
