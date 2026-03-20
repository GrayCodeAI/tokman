package filter

// ByteSlicePool provides reusable byte slices to reduce GC pressure.
// T89: Pre-allocate and reuse buffers in hot path.
type ByteSlicePool struct {
	pool chan []byte
	size int
}

// NewByteSlicePool creates a pool of byte slices of given size.
func NewByteSlicePool(poolSize, sliceSize int) *ByteSlicePool {
	return &ByteSlicePool{
		pool: make(chan []byte, poolSize),
		size: sliceSize,
	}
}

// Get retrieves a byte slice from the pool or creates a new one.
func (p *ByteSlicePool) Get() []byte {
	select {
	case buf := <-p.pool:
		return buf[:0]
	default:
		return make([]byte, 0, p.size)
	}
}

// Put returns a byte slice to the pool.
func (p *ByteSlicePool) Put(buf []byte) {
	if cap(buf) < p.size/2 || cap(buf) > p.size*4 {
		return // Don't pool slices that are too small or too large
	}
	select {
	case p.pool <- buf[:0]:
	default:
		// Pool full, discard
	}
}

// LineScanner provides zero-allocation line scanning.
// T89: Avoid strings.Split allocation in hot path.
type LineScanner struct {
	data []byte
	pos  int
}

// NewLineScanner creates a scanner over byte data.
func NewLineScanner(data []byte) *LineScanner {
	return &LineScanner{data: data, pos: 0}
}

// Next returns the next line without trailing newline.
// Returns nil when no more lines.
func (s *LineScanner) Next() []byte {
	if s.pos >= len(s.data) {
		return nil
	}

	start := s.pos
	for s.pos < len(s.data) && s.data[s.pos] != '\n' {
		s.pos++
	}

	line := s.data[start:s.pos]

	// Skip the newline
	if s.pos < len(s.data) && s.data[s.pos] == '\n' {
		s.pos++
	}

	// Trim trailing \r
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}

	return line
}

// Remaining returns bytes remaining after current position.
func (s *LineScanner) Remaining() []byte {
	if s.pos >= len(s.data) {
		return nil
	}
	return s.data[s.pos:]
}

// CountLines counts newlines in data without allocating.
func CountLines(data []byte) int {
	count := 0
	for _, b := range data {
		if b == '\n' {
			count++
		}
	}
	// If data doesn't end with newline, count last line
	if len(data) > 0 && data[len(data)-1] != '\n' {
		count++
	}
	return count
}
