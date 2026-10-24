package ownedfile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"slices"
)

const hashTag = " sha256="

func Stamp(body []byte) []byte {
	at := bytes.Index(body, []byte(Marker))
	if at < 0 {
		return body
	}
	end := at + len(Marker)
	sum := sha256.Sum256(body)
	return slices.Concat(body[:end], []byte(hashTag+hex.EncodeToString(sum[:])), body[end:])
}

func Edited(content []byte) bool {
	at := bytes.Index(content, []byte(Marker))
	if at < 0 {
		return false
	}
	start := at + len(Marker)
	rest := content[start:]
	if !bytes.HasPrefix(rest, []byte(hashTag)) {
		// Written before bodies carried a hash: nothing to compare against.
		return false
	}
	tokenEnd := len(hashTag) + sha256.Size*2
	if len(rest) < tokenEnd {
		return true
	}
	want := string(rest[len(hashTag):tokenEnd])
	sum := sha256.Sum256(slices.Concat(content[:start], rest[tokenEnd:]))
	return hex.EncodeToString(sum[:]) != want
}

func EditedMessage(path, backupPath string) string {
	return "kept your edits of " + path + " in " + backupPath
}
