# EXPECT:
# counter? true
# string? true
# wrong? false
# 0

class Counter {
}

def main() Int {
	value = Counter()
	Term.println("counter? " + (value is Counter))
	Term.println("string? " + ("hello" is String))
	Term.println("wrong? " + (value is String))
	0
}
