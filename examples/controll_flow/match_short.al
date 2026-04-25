# EXPECT:
# pair 5-9
# 14
# worker 7
# 0

class PairBox {
    left Int
    right Int
}

interface WorkerLike {
    def doWork() Int
}

class Worker with WorkerLike {
    def doWork() Int = 7
}

def main() Int {
    pair PairBox = PairBox(5, 9)

    match pair: PairBox(left, right) => Term.println("pair " + left + "-" + right)

    picked = match pair: PairBox(left, right) => left + right
    Term.println(picked)

    workerLike WorkerLike = Worker()
    match workerLike: worker Worker => Term.println("worker " + worker.doWork())

    0
}
