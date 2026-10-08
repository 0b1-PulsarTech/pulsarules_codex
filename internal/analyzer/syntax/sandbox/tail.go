package sandbox

import (
	"strconv"
	"sync"
)

const stderrTail = 2 << 10

type tail struct {
	mu     sync.Mutex
	recent []byte
}

func (buffer *tail) Write(chunk []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	buffer.recent = append(buffer.recent, chunk...)
	if extra := len(buffer.recent) - stderrTail; extra > 0 {
		kept := copy(buffer.recent, buffer.recent[extra:])
		buffer.recent = buffer.recent[:kept]
	}
	return len(chunk), nil
}

func (buffer *tail) String() string {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	if len(buffer.recent) == 0 {
		return ""
	}
	return strconv.QuoteToASCII(string(buffer.recent))
}
