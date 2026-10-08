package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

type unicodeMeta struct {
	Version  float64
	Group    string
	Subgroup string
}

func fetchShortcodes() (map[string]string, error) {
	const shortcodeURL = "https://api.github.com/emojis"

	body, err := get(shortcodeURL)
	if err != nil {
		return nil, err
	}

	var raw map[string]string
	if unmarshalErr := json.Unmarshal(body, &raw); unmarshalErr != nil {
		return nil, fmt.Errorf("decode shortcodes: %w", unmarshalErr)
	}

	const marker = "/unicode/"
	codepoints := make(map[string]string, len(raw))
	for name, iconURL := range raw {
		// GitHub's proprietary images (octocat, trollface, shipit, dependabot) have no
		// /unicode/ segment, so dropping the segment-less ones filters them out.
		_, rest, found := strings.Cut(iconURL, marker)
		if !found {
			continue
		}
		if trimmed, ok := strings.CutSuffix(firstField(rest, "?"), ".png"); ok {
			codepoints[name] = strings.ToLower(trimmed)
		}
	}
	return codepoints, nil
}

func fetchUnicodeMeta() (map[string]unicodeMeta, error) {
	const unicodeURL = "https://unicode.org/Public/emoji/latest/emoji-test.txt"
	const estimatedSequenceCount = 4096
	const (
		scannerStartBufferBytes = 64 * 1024
		scannerMaxBufferBytes   = 1024 * 1024
	)

	body, err := get(unicodeURL)
	if err != nil {
		return nil, err
	}

	meta := make(map[string]unicodeMeta, estimatedSequenceCount)
	var group, subgroup string
	scanner := bufio.NewScanner(bytes.NewReader(body))
	// Well above emoji-test.txt's longest line, so bufio.Scanner never reports ErrTooLong.
	scanner.Buffer(make([]byte, 0, scannerStartBufferBytes), scannerMaxBufferBytes)
	for scanner.Scan() {
		line := scanner.Text()
		if after, ok := strings.CutPrefix(line, "# group:"); ok {
			group = strings.TrimSpace(after)
			continue
		}
		if after, ok := strings.CutPrefix(line, "# subgroup:"); ok {
			subgroup = strings.TrimSpace(after)
			continue
		}
		key, version, ok := parseMetaLine(line)
		if !ok {
			continue
		}
		sequenceMeta := unicodeMeta{Version: version, Group: group, Subgroup: subgroup}
		meta[key] = sequenceMeta
		// GitHub's icon URLs drop the zero-width joiner and the variation
		// selector, so a joined sequence like pirate_flag only matches on a
		// key with both stripped. First spelling wins.
		if joinless := stripJoiners(key); joinless != key {
			if _, taken := meta[joinless]; !taken {
				meta[joinless] = sequenceMeta
			}
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf("scan unicode data: %w", scanErr)
	}
	return meta, nil
}

func parseMetaLine(line string) (string, float64, bool) {
	if strings.HasPrefix(line, "#") {
		return "", 0, false
	}
	codepoints, rest, found := strings.Cut(line, ";")
	// Only fully-qualified lines count; the others are variation-selector spellings.
	if !found || !strings.Contains(rest, "fully-qualified") {
		return "", 0, false
	}
	_, comment, found := strings.Cut(rest, "#")
	if !found {
		return "", 0, false
	}
	version, ok := parseVersion(comment)
	if !ok {
		return "", 0, false
	}
	return strings.ToLower(strings.Join(strings.Fields(codepoints), "-")), version, true
}

func parseVersion(comment string) (float64, bool) {
	for field := range strings.FieldsSeq(comment) {
		digits, ok := strings.CutPrefix(field, "E")
		if !ok {
			continue
		}
		if value, err := strconv.ParseFloat(digits, 64); err == nil {
			return value, true
		}
	}
	return 0, false
}

func stripJoiners(key string) string {
	parts := slices.DeleteFunc(strings.Split(key, "-"), func(part string) bool {
		return part == "200d" || part == "fe0f"
	})
	return strings.Join(parts, "-")
}

func firstField(value, sep string) string {
	head, _, _ := strings.Cut(value, sep)
	return head
}

func get(url string) ([]byte, error) {
	const fetchTimeout = 60 * time.Second

	client := &http.Client{Timeout: fetchTimeout}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request %s: %w", url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get %s: status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", url, err)
	}
	return body, nil
}
