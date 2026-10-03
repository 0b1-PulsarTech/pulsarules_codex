package movepurity

type gitReader interface {
	HeadBlob(path string) (string, error)
	StagedBlob(path string) (string, error)
	HeadEntry(path string) (mode, id string, err error)
	StagedEntry(path string) (mode, id string, err error)
	HeadDir(dir string) ([]string, error)
	StagedDir(dir string) ([]string, error)
}

type (
	blobResult struct {
		content string
		err     error
	}
	cachedReader struct {
		gitReader
		head   map[string]blobResult
		staged map[string]blobResult
	}
)

func newCachedReader(inner gitReader) *cachedReader {
	return &cachedReader{
		gitReader: inner,
		head:      map[string]blobResult{},
		staged:    map[string]blobResult{},
	}
}

func (reader *cachedReader) HeadBlob(path string) (string, error) {
	return lookupBlob(reader.head, path, reader.gitReader.HeadBlob)
}

func (reader *cachedReader) StagedBlob(path string) (string, error) {
	return lookupBlob(reader.staged, path, reader.gitReader.StagedBlob)
}

func lookupBlob(
	cache map[string]blobResult, path string, read func(string) (string, error),
) (string, error) {
	if hit, ok := cache[path]; ok {
		return hit.content, hit.err
	}
	content, err := read(path)
	cache[path] = blobResult{content: content, err: err}
	return content, err
}
