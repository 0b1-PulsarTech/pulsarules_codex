package dartsyntax

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/execx"
)

func parseSDKVersion(output string) (string, error) {
	match := regexp.MustCompile(`Dart (?:SDK|VM) version:\s*(\S+)`).FindStringSubmatch(output)
	if match == nil {
		return "", fmt.Errorf("dart --version: no version in %q", tail([]byte(output)))
	}
	return match[1], nil
}

func checkSDK(version string) error {
	const (
		minSDKMajor = 3
		minSDKMinor = 11
		prerelease  = 4
	)
	floor := []int{minSDKMajor, minSDKMinor, 0}
	match := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(-\S+)?$`).FindStringSubmatch(version)
	if match != nil {
		parts := make([]int, len(floor))
		for index := range parts {
			parts[index], _ = strconv.Atoi(match[index+1])
		}
		// A prerelease sorts below its release, so pub's ^3.11.0 excludes 3.11.0-x.
		order := slices.Compare(parts, floor)
		if order > 0 || (order == 0 && match[prerelease] == "") {
			return nil
		}
	}
	const message = "check dart version: Dart SDK %d.%d+ required, found %s"
	return fmt.Errorf(message, minSDKMajor, minSDKMinor, version)
}

func sdkVersion(ctx context.Context, dart string) (string, error) {
	result, err := execx.Run(ctx, execx.Command{Name: dart, Args: dartArgs([]string{"--version"})})
	if err != nil {
		return "", stepError(ctx, "--version", result, err)
	}
	version, err := parseSDKVersion(result.Stdout + "\n" + result.Stderr)
	if err != nil {
		return "", err
	}
	if checkErr := checkSDK(version); checkErr != nil {
		return "", checkErr
	}
	return version, nil
}

func dartArgs(args []string) []string {
	return append([]string{"--suppress-analytics"}, args...)
}

func stepError(ctx context.Context, step string, result execx.Result, err error) error {
	if execErr, ok := errors.AsType[*execx.Error](err); ok {
		err = execErr.Err
	}
	if ctx.Err() != nil && !errors.Is(err, ctx.Err()) {
		err = fmt.Errorf("%w: %w", ctx.Err(), err)
	}
	output := tail([]byte(result.Stderr + "\n" + result.Stdout))
	if output == "" {
		return fmt.Errorf("dart %s: %w", step, err)
	}
	return fmt.Errorf("dart %s: %w: %s", step, err, output)
}

func tail(output []byte) string {
	const maxChars = 400
	text := strings.TrimSpace(string(output))
	if len(text) <= maxChars {
		return text
	}
	start := len(text) - maxChars
	for start < len(text) && !utf8.RuneStart(text[start]) {
		start++
	}
	return text[start:]
}
