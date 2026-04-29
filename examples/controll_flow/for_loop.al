# EXPECT:
# item 1
# item 2
# item 3
# total 6
# 0

def main() Int {
	total Int := 0

	for item <- [1, 2, 3] {
		OS.println("item " + item)
		total += item
	}

	OS.println("total " + total)
	0
}
