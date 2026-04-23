# EXPECT:
# value is Counter? true
# counter is CounterPrecursor? true
# string is Str? true
# value is Str? false
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
	Term.println("string is Str? " + ("hello" is Str))
	Term.println("value is Str? " + (counter is Str))
	Term.println("counter is Breaker? " + (counter is Breaker))
	0
}
