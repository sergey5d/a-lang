# EXPECT:
# plain function regular-7
# named function named-9
# regular method method-11
# named method method-13
# 0

class Formatter {
	def format(prefix String, value Int) String {
		return prefix + value
	}
}

def format(prefix String, value Int) String {
	return prefix + value
}

def main() Int {
	formatter = Formatter()

	Term.println("plain function", format("regular-", 7))
	Term.println("named function", format(value = 9, prefix = "named-"))
	Term.println("regular method", formatter.format("method-", 11))
	Term.println("named method", formatter.format(value = 13, prefix = "method-"))

	0
}
