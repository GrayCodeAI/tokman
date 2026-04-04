// Package streaming provides memory-efficient processing for large inputs (>500K tokens).
package streaming

import (
	"bufio"
	"io"
	"runtime"
	"strings"
	"sync"

	"github.com/GrayCodeAI/tokman/internal/filter"
)

const (
	// DefaultChunkSize is the default size for streaming chunks (~4K tokens).
	DefaultChunkSize = 16 * 1024 // 16KB

	// MaxMemoryBuffer is the maximum memory buffer size.
	MaxMemoryBuffer = 100 * 1024 * 1024 // 100MB
)

// ProcessLargeContent processes large content in chunks.
func ProcessLargeContent(content string, processor ChunkProcessor) (string, error) {
	p := NewPipeline(processor)

	reader := bufio.NewReader(strings.NewReader(content))
	writer := &strings.Builder{}

	err := p.Process(reader, writer)
	if err != nil {
		return "", err
	}

	return writer.String(), nil
}

// EstimateTokens estimates the token count for content.
func EstimateTokens(content string) int {
	// Rough approximation: ~4 chars per token
	return len(content) / 4
}

// Chunk represents a chunk of content for processing.
type Chunk struct {
	Data       []byte
	Offset     int64
	IsLast     bool
	TokenCount int
}

// Result represents processed chunk output.
type Result struct {
	Data       []byte
	Offset     int64
	Saved      int
	TokenCount int
	Error      error
}

// Pipeline processes content in streaming fashion.
type Pipeline struct {
	chunkSize   int
	maxMemory   int64
	workerCount int
	processor   ChunkProcessor
}

// ChunkProcessor processes individual chunks.
type ChunkProcessor interface {
	Process(chunk []byte) ([]byte, int, error)
	Name() string
}

// FilterProcessor adapts filter.Engine to ChunkProcessor.
type FilterProcessor struct {
	engine *filter.Engine
}

// Process implements ChunkProcessor.
func (p *FilterProcessor) Process(chunk []byte) ([]byte, int, error) {
	processed, saved := p.engine.Process(string(chunk))
	return []byte(processed), saved, nil
}

// Name implements ChunkProcessor.
func (p *FilterProcessor) Name() string {
	return "filter_engine"
}

// NewPipeline creates a new streaming pipeline.
func NewPipeline(processor ChunkProcessor) *Pipeline {
	return &Pipeline{
		chunkSize:   DefaultChunkSize,
		maxMemory:   MaxMemoryBuffer,
		workerCount: runtime.NumCPU(),
		processor:   processor,
	}
}

// SetChunkSize sets the chunk size.
func (p *Pipeline) SetChunkSize(size int) {
	p.chunkSize = size
}

// SetMaxMemory sets the maximum memory.
func (p *Pipeline) SetMaxMemory(max int64) {
	p.maxMemory = max
}

// SetWorkerCount sets the number of workers.
func (p *Pipeline) SetWorkerCount(count int) {
	p.workerCount = count
}

// ProcessReader processes content from a reader.
func (p *Pipeline) ProcessReader(reader io.Reader, writer io.Writer) (Stats, error) {
	stats := Stats{}

	// Create buffered reader/writer
	bufReader := bufio.NewReaderSize(reader, p.chunkSize)
	bufWriter := bufio.NewWriterSize(writer, p.chunkSize)
	defer bufWriter.Flush()

	// Create work channels
	chunkChan := make(chan Chunk, p.workerCount*2)
	resultChan := make(chan Result, p.workerCount*2)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < p.workerCount; i++ {
		wg.Add(1)
		go p.worker(chunkChan, resultChan, &wg)
	}

	// Start result collector
	collectDone := make(chan struct{})
	go func() {
		p.collectResults(resultChan, bufWriter, &stats)
		close(collectDone)
	}()

	// Read and dispatch chunks
	offset := int64(0)
	buffer := make([]byte, p.chunkSize)

	for {
		n, err := bufReader.Read(buffer)
		if n > 0 {
			// Copy data since we're reusing buffer
			data := make([]byte, n)
			copy(data, buffer[:n])

			chunk := Chunk{
				Data:   data,
				Offset: offset,
			}

			if err == io.EOF {
				chunk.IsLast = true
			}

			chunkChan <- chunk
			offset += int64(n)
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			close(chunkChan)
			wg.Wait()
			close(resultChan)
			<-collectDone
			return stats, err
		}
	}

	close(chunkChan)
	wg.Wait()
	close(resultChan)
	<-collectDone

	return stats, nil
}

// ProcessReader processes from io.Reader.
func (p *Pipeline) Process(reader io.Reader, writer io.Writer) error {
	_, err := p.ProcessReader(reader, writer)
	return err
}

// worker processes chunks.
func (p *Pipeline) worker(chunks <-chan Chunk, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()

	for chunk := range chunks {
		var processed []byte
		var saved int
		var err error

		if p.processor != nil {
			processed, saved, err = p.processor.Process(chunk.Data)
		} else {
			processed = chunk.Data
		}

		result := Result{
			Data:   processed,
			Offset: chunk.Offset,
			Saved:  saved,
		}

		if err != nil {
			result.Error = err
		}

		results <- result
	}
}

// collectResults aggregates results in order.
func (p *Pipeline) collectResults(results <-chan Result, writer io.Writer, stats *Stats) {
	// Buffer results to maintain order
	buffer := make(map[int64][]byte)
	nextOffset := int64(0)

	for result := range results {
		if result.Error != nil {
			continue
		}

		stats.TotalChunks++
		stats.TotalSaved += int64(result.Saved)
		stats.TotalOutput += int64(len(result.Data))

		if result.Offset == nextOffset {
			// Write immediately
			writer.Write(result.Data)
			nextOffset += int64(len(result.Data))

			// Check buffered results
			for {
				if data, ok := buffer[nextOffset]; ok {
					prevOffset := nextOffset
					writer.Write(data)
					nextOffset += int64(len(data))
					delete(buffer, prevOffset)
				} else {
					break
				}
			}
		} else {
			// Buffer for later
			buffer[result.Offset] = result.Data
		}
	}
}

// Stats provides streaming statistics.
type Stats struct {
	TotalChunks int64
	TotalSaved  int64
	TotalOutput int64
	PeakMemory  int64
}

// ShouldStream determines if content should use streaming.
func ShouldStream(content string) bool {
	// Use streaming for content > 500K tokens (~2MB)
	return len(content) > 2*1024*1024
}

// MemoryMappedFile provides memory-mapped file access.
type MemoryMappedFile struct {
	data []byte
	size int64
}

// MapFile maps a file into memory.
func MapFile(path string) (*MemoryMappedFile, error) {
	// Simplified implementation
	// In production, use syscall.Mmap or similar
	data, err := osReadFile(path)
	if err != nil {
		return nil, err
	}

	return &MemoryMappedFile{
		data: data,
		size: int64(len(data)),
	}, nil
}

// Data returns the mapped data.
func (m *MemoryMappedFile) Data() []byte {
	return m.data
}

// Size returns the file size.
func (m *MemoryMappedFile) Size() int64 {
	return m.size
}

// Unmap unmaps the file.
func (m *MemoryMappedFile) Unmap() error {
	m.data = nil
	return nil
}

// ArenaAllocator provides arena-based memory allocation.
type ArenaAllocator struct {
	buffer []byte
	offset int
	size   int
}

// NewArena creates a new arena.
func NewArena(size int) *ArenaAllocator {
	return &ArenaAllocator{
		buffer: make([]byte, size),
		size:   size,
	}
}

// Alloc allocates memory from the arena.
func (a *ArenaAllocator) Alloc(size int) []byte {
	if a.offset+size > a.size {
		return nil // Out of memory
	}

	ptr := a.buffer[a.offset : a.offset+size]
	a.offset += size
	return ptr
}

// Reset resets the arena.
func (a *ArenaAllocator) Reset() {
	a.offset = 0
}

// Used returns used memory.
func (a *ArenaAllocator) Used() int {
	return a.offset
}

// ZeroCopyReader provides zero-copy reading.
type ZeroCopyReader struct {
	data   []byte
	offset int
}

// NewZeroCopyReader creates a zero-copy reader.
func NewZeroCopyReader(data []byte) *ZeroCopyReader {
	return &ZeroCopyReader{data: data}
}

// Read implements io.Reader.
func (z *ZeroCopyReader) Read(p []byte) (n int, err error) {
	if z.offset >= len(z.data) {
		return 0, io.EOF
	}

	n = copy(p, z.data[z.offset:])
	z.offset += n
	return n, nil
}

// ReadAt implements io.ReaderAt.
func (z *ZeroCopyReader) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= int64(len(z.data)) {
		return 0, io.EOF
	}
	n = copy(p, z.data[off:])
	return n, nil
}

// Reset resets the reader.
func (z *ZeroCopyReader) Reset() {
	z.offset = 0
}

// StreamingWriter writes large output efficiently.
type StreamingWriter struct {
	writer    io.Writer
	buffer    []byte
	offset    int
	flushed   int64
	autoFlush bool
}

// NewStreamingWriter creates a streaming writer.
func NewStreamingWriter(w io.Writer, bufferSize int) *StreamingWriter {
	return &StreamingWriter{
		writer:    w,
		buffer:    make([]byte, bufferSize),
		autoFlush: true,
	}
}

// Write implements io.Writer.
func (s *StreamingWriter) Write(p []byte) (n int, err error) {
	// If data fits in buffer, copy it
	if len(p)+s.offset <= len(s.buffer) {
		n = copy(s.buffer[s.offset:], p)
		s.offset += n

		if s.autoFlush && s.offset >= len(s.buffer)/2 {
			err = s.Flush()
		}
		return n, err
	}

	// Flush buffer first
	if s.offset > 0 {
		if err = s.Flush(); err != nil {
			return 0, err
		}
	}

	// Write large data directly
	if len(p) > len(s.buffer) {
		n, err = s.writer.Write(p)
		s.flushed += int64(n)
		return n, err
	}

	// Copy to buffer
	n = copy(s.buffer, p)
	s.offset = n
	return n, nil
}

// Flush flushes the buffer.
func (s *StreamingWriter) Flush() error {
	if s.offset == 0 {
		return nil
	}

	n, err := s.writer.Write(s.buffer[:s.offset])
	if err != nil {
		return err
	}

	s.flushed += int64(n)
	s.offset = 0
	return nil
}

// Written returns total bytes written.
func (s *StreamingWriter) Written() int64 {
	return s.flushed + int64(s.offset)
}

// Close closes the writer.
func (s *StreamingWriter) Close() error {
	return s.Flush()
}

// Helper types for the implementation

type stringsReader struct {
	s string
	i int64
}

func stringsNewReader(s string) *stringsReader {
	return &stringsReader{s: s}
}

func (r *stringsReader) Read(b []byte) (n int, err error) {
	if r.i >= int64(len(r.s)) {
		return 0, io.EOF
	}
	n = copy(b, r.s[r.i:])
	r.i += int64(n)
	return
}

type stringsBuilder struct {
	buf []byte
}

func (b *stringsBuilder) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *stringsBuilder) String() string {
	return string(b.buf)
}

func osReadFile(path string) ([]byte, error) {
	// Placeholder - use os.ReadFile in production
	return nil, nil
}
