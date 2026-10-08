package sandbox

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
)

const (
	headerBytes = 4
	maxFrame    = 16 << 20
)

var errFrameSize = errors.New("frame is over the size limit")

func writeFrame(out io.Writer, payload any) error {
	var body bytes.Buffer
	if err := gob.NewEncoder(&body).Encode(payload); err != nil {
		return fmt.Errorf("encode frame: %w", err)
	}
	if body.Len() > maxFrame {
		return fmt.Errorf("encode frame: %w", errFrameSize)
	}
	frame := make([]byte, headerBytes, headerBytes+body.Len())
	//nolint:gosec // The size is bounded by maxFrame above.
	binary.BigEndian.PutUint32(frame, uint32(body.Len()))
	if _, err := out.Write(append(frame, body.Bytes()...)); err != nil {
		return fmt.Errorf("write frame: %w", err)
	}
	return nil
}

func readFrame(in io.Reader, into any) error {
	var header [headerBytes]byte
	if _, err := io.ReadFull(in, header[:]); err != nil {
		return fmt.Errorf("read frame header: %w", err)
	}
	size := binary.BigEndian.Uint32(header[:])
	if size > maxFrame {
		return fmt.Errorf("read frame: %w", errFrameSize)
	}
	body := make([]byte, size)
	if _, err := io.ReadFull(in, body); err != nil {
		return fmt.Errorf("read frame body: %w", noEOF(err))
	}
	if err := gob.NewDecoder(bytes.NewReader(body)).Decode(into); err != nil {
		return fmt.Errorf("decode frame: %w", err)
	}
	return nil
}

func noEOF(err error) error {
	if errors.Is(err, io.EOF) {
		return io.ErrUnexpectedEOF
	}
	return err
}
