# EXPECT:
# set 1
# set 2
# set 3
# pair a 1
# pair b 2
# doubled true 3
# expanded true 6
# filtered true 2
# setFold 6
# setReduce 6
# setExists true
# setForAll true
# mapped 10 20
# expandedValues 1 12
# filteredMap true 1
# mapFold 3
# mapReduce b 2
# mapExists true
# mapForAll true
# total 9

def main() {
    seen = Set(1, 2, 3)
    doubled = seen.map((item Int) -> item * 2)
    expanded = seen.flatMap((item Int) -> Set(item, item + 10))
    filtered = seen.filter((item Int) -> item > 1)
    setTotal = seen.fold(0, (acc Int, item Int) -> acc + item)
    setReduced = seen.reduce((left Int, right Int) -> left + right)
    setHasBig = seen.exists((item Int) -> item > 2)
    setAllPositive = seen.forAll((item Int) -> item > 0)
    seen.forEach((item Int) -> OS.println("set " + item))

    values = Map("a" : 1, "b" : 2)
    mapped = values.map((key Str, value Int) -> value * 10)
    expandedValues = values.flatMap((key Str, value Int) -> List(value, value + 10))
    filteredMap = values.filter((key Str, value Int) -> value > 1)
    mapTotal = values.fold(0, (acc Int, key Str, value Int) -> acc + value)
    reducedPair = values.reduce((leftKey Str, leftValue Int, rightKey Str, rightValue Int) -> (rightKey, rightValue)).get()
    reducedKey, reducedValue = reducedPair
    mapHasB = values.exists((key Str, value Int) -> key == "b")
    mapAllSmall = values.forAll((key Str, value Int) -> value < 3)
    values.forEach((key Str, value Int) -> OS.println("pair " + key + " " + value))

    OS.println("doubled " + doubled.contains(4) + " " + doubled.size())
    OS.println("expanded " + expanded.contains(12) + " " + expanded.size())
    OS.println("filtered " + filtered.contains(2) + " " + filtered.size())
    OS.println("setFold " + setTotal)
    OS.println("setReduce " + setReduced.getOr(0))
    OS.println("setExists " + setHasBig)
    OS.println("setForAll " + setAllPositive)
    OS.println("mapped " + mapped.get(0).getOr(0) + " " + mapped.get(1).getOr(0))
    OS.println("expandedValues " + expandedValues.get(0).getOr(0) + " " + expandedValues.get(3).getOr(0))
    OS.println("filteredMap " + filteredMap.contains("b") + " " + filteredMap.size())
    OS.println("mapFold " + mapTotal)
    OS.println("mapReduce " + reducedKey + " " + reducedValue)
    OS.println("mapExists " + mapHasB)
    OS.println("mapForAll " + mapAllSmall)

    total := 0
    for item Int <- seen {
        total += item
    }
    for key Str, value Int <- values {
        total += value
    }
    OS.println("total " + total)
}
