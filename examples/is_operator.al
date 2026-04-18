# EXPECT:
# value is Counter? true
# counter is CounterPrecursor? true
# string is String? true
# value is String? false
# counter is Breaker? false
# 0

interface CounterPrecursor {
}

class Counter with CounterPrecursor {
}

interface Breaker {
}

def main() Int {
	counter = Counter()
	Term.println("value is Counter? " + (counter is Counter))
	Term.println("counter is CounterPrecursor? " + (counter is CounterPrecursor))
	Term.println("string is String? " + ("hello" is String))
	Term.println("value is String? " + (counter is String))
	Term.println("counter is Breaker? " + (counter is Breaker))
	0
}
