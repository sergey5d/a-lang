# EXPECT:
# Ana is 10
# Ben is 12
# 1
# 20

def describe(user { name Str, age Int }) Str =
    user.name + " is " + user.age

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

    OS.println(describe(full))
    OS.println(describe(smaller))
    OS.println(inferred.count)
    OS.println(mixed.a + mixed.b + mixed.c)
}
