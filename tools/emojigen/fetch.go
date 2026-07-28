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

const (
	shortcodeURL = "https://api.github.com/emojis"
	unicodeURL   = "https://unicode.org/Public/emoji/latest/emoji-test.txt"
	fetchTimeout = 60 * time.Second
)

// unicodeMeta is the Unicode registry's view of one emoji sequence.
type unicodeMeta struct {
	Version  float64
	Group    string
	Subgroup string
}

// GitHub's proprietary images (octocat, trollface, shipit, dependabot) have no
// /unicode/ segment in their URL; dropping the segment-less ones is the whole
// filter for them.
func fetchShortcodes() (map[string]string, error) {
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
	body, err := get(unicodeURL)
	if err != nil {
		return nil, err
	}

	meta := make(map[string]unicodeMeta, 4096)
	var group, subgroup string
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
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
		info := unicodeMeta{Version: version, Group: group, Subgroup: subgroup}
		meta[key] = info
		// GitHub's icon URLs drop the zero-width joiner and the variation
		// selector, so a joined sequence like pirate_flag only matches on a
		// key with both stripped. First spelling wins.
		if joinless := stripJoiners(key); joinless != key {
			if _, taken := meta[joinless]; !taken {
				meta[joinless] = info
			}
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf("scan unicode data: %w", scanErr)
	}
	return meta, nil
}

// A fully-qualified emoji-test.txt data line reads:
//
//	1F600 ; fully-qualified  # <emoji> E1.0 grinning face
//
// Non-fully-qualified lines are variation-selector spellings of the same emoji.
func parseMetaLine(line string) (string, float64, bool) {
	if strings.HasPrefix(line, "#") {
		return "", 0, false
	}
	codepoints, rest, found := strings.Cut(line, ";")
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
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request %s: %w", url, err)
	}
	resp, err := http.DefaultClient.Do(req)
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
