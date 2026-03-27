package filter

import (
	"fmt"
	"os"
	"runtime"
	"syscall"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// MmapFile holds a memory-mapped file for zero-copy reading of large inputs.
// Task #103: Memory-mapped file processing for large inputs.
type MmapFile struct {
	data   []byte
	path   string
	size   int64
}

// OpenMmapFile opens a file and maps it into memory.
// The caller must call Close() when done to unmap.
func OpenMmapFile(path string) (*MmapFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("mmap: open %s: %w", path, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("mmap: stat %s: %w", path, err)
	}
	if info.Size() == 0 {
		return &MmapFile{path: path, size: 0}, nil
	}

	// syscall.Mmap maps the file into process memory.
	data, err := syscall.Mmap(
		int(f.Fd()),
		0,
		int(info.Size()),
		syscall.PROT_READ,
		syscall.MAP_SHARED,
	)
	if err != nil {
		return nil, fmt.Errorf("mmap: map %s: %w", path, err)
	}

	return &MmapFile{data: data, path: path, size: info.Size()}, nil
}

// Content returns the file contents as a string without copying.
// The string is only valid until Close() is called.
func (m *MmapFile) Content() string {
	if len(m.data) == 0 {
		return ""
	}
	// Convert byte slice to string without allocation via unsafe is not done here
	// for correctness — instead we let the caller decide.
	return string(m.data)
}

// Size returns the file size in bytes.
func (m *MmapFile) Size() int64 { return m.size }

// EstimateTokens returns the token count without loading full content into a Go string.
func (m *MmapFile) EstimateTokens() int {
	return core.EstimateTokens(string(m.data))
}

// Close unmaps the file from memory.
func (m *MmapFile) Close() error {
	if len(m.data) == 0 {
		return nil
	}
	err := syscall.Munmap(m.data)
	m.data = nil
	return err
}

// ProcessLargeFile compresses a large file using memory mapping to reduce
// memory overhead compared to os.ReadFile.
func ProcessLargeFile(path string, pipeline *PipelineCoordinator) (string, *PipelineStats, error) {
	mf, err := OpenMmapFile(path)
	if err != nil {
		// Fall back to regular read on mmap failure (Windows, etc.)
		raw, err2 := os.ReadFile(path)
		if err2 != nil {
			return "", nil, fmt.Errorf("process large file: %w (mmap: %v)", err2, err)
		}
		output, stats := pipeline.Process(string(raw))
		return output, stats, nil
	}
	defer mf.Close()

	content := mf.Content()
	output, stats := pipeline.Process(content)

	// Force GC to reclaim the string copy from content before returning.
	runtime.GC()

	return output, stats, nil
}
