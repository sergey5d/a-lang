# EXPECT:
# 7
# 0

interface WorkerLike {
    def doWork() Int
}

class Worker with WorkerLike {
}

impl Worker {
    impl def doWork() Int = 7
}

class Other with WorkerLike {
}

impl Other {
    impl def doWork() Int = 3
}

class PairBox {
    left Int
    right Int
}

def main() Int {
    workerLike WorkerLike = Worker()
    OS.println(match workerLike {
        worker Worker => worker.doWork()
        _ Other => 100
        _ => 0
    })

    pair PairBox = PairBox(4, 9)
    result = match pair {
        PairBox(left, right) => left + right
    }
    if result != 13 {
        return 1
    }

    0
}
