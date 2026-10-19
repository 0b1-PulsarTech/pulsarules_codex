package dartsyntax

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/process"
)

type request struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Test    bool   `json:"test"`
}

func collectRequests(sources []syntax.Source) []request {
	requests := make([]request, 0, len(sources))
	for _, source := range sources {
		requests = append(requests, request{
			Path: source.Path, Content: string(source.Content), Test: source.IsTest,
		})
	}
	return requests
}

func runHelper(
	ctx context.Context, exe, tempDir string, requests []request,
) (map[string]*syntax.File, error) {
	payload, err := json.Marshal(requests)
	if err != nil {
		return nil, fmt.Errorf("encode helper request: %w", err)
	}
	requestPath, err := writeRequest(tempDir, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	result, runErr := process.Run(ctx, process.Command{Name: exe, Args: []string{requestPath}})
	cleanErr := removeRequest(requestPath)
	if runErr != nil {
		return nil, errors.Join(stepError(ctx, "helper run", result, runErr), cleanErr)
	}
	files, decodeErr := decodeFiles([]byte(result.Stdout), requests)
	return files, errors.Join(decodeErr, cleanErr)
}

func writeRequest(tempDir string, payload io.WriterTo) (string, error) {
	file, err := os.CreateTemp(tempDir, "pulsarules-dart-request-*.json")
	if err != nil {
		return "", fmt.Errorf("create helper request: %w", err)
	}
	_, writeErr := payload.WriteTo(file)
	if joined := errors.Join(writeErr, file.Close()); joined != nil {
		wrapped := fmt.Errorf("write helper request: %w", joined)
		return "", errors.Join(wrapped, removeRequest(file.Name()))
	}
	return file.Name(), nil
}

func removeRequest(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove helper request: %w", err)
	}
	return nil
}

func decodeFiles(raw []byte, requests []request) (map[string]*syntax.File, error) {
	var decoded map[string]*syntax.File
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("decode dart helper output: %w", err)
	}
	files := make(map[string]*syntax.File, len(requests))
	for _, item := range requests {
		file := decoded[item.Path]
		if file == nil {
			continue
		}
		file.Path = item.Path
		file.Language = language.Dart
		files[item.Path] = file
	}
	return files, nil
}
