# EXPECT:
# plain function regular-7
# mixed function named-9
# regular method method-11
# mixed method method-13
# named function named-15
# named method method-17
# 0

class Formatter {
	def format(prefix Str, value Int) Str {
		return prefix + value
	}
}

def format(prefix Str, value Int) Str {
	return prefix + value
}

def main() Int {
	formatter = Formatter()

	Term.println("plain function", format("regular-", 7))
	Term.println("mixed function", format("named-", value = 9))
	Term.println("regular method", formatter.format("method-", 11))
	Term.println("mixed method", formatter.format("method-", value = 13))
	Term.println("named function", format(value = 15, prefix = "named-"))
	Term.println("named method", formatter.format(value = 17, prefix = "method-"))

	0
}
