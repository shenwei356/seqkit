package cmd

// orderedOutputBatches bounds admitted work while allowing workers to overlap
// with output and short delays in earlier records.
const orderedOutputBatches = 4

// consumeOrdered emits consecutive results as soon as they are ready. IDs must
// be unique and start at one. Callers bound pending results by holding admission
// tokens until emit returns, including results that produce no output.
func consumeOrdered[T any](results <-chan T, id func(T) uint64, emit func(T)) {
	next := uint64(1)
	pending := make(map[uint64]T)
	for result := range results {
		if current := id(result); current != next {
			pending[current] = result
			continue
		}
		emit(result)
		next++
		for {
			result, ok := pending[next]
			if !ok {
				break
			}
			delete(pending, next)
			emit(result)
			next++
		}
	}
}
