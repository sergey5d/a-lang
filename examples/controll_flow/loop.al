# EXPECT:
# loop 0
# loop 1
# loop 2
# 0

def main() Int {
	counter Int := 0

	loop {
		OS.println("loop " + counter)
		counter += 1
		if counter == 3 {
			break
		}
	}

	0
}
