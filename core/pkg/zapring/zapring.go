package zapring

// in-memory ring buffer log

import (
	"io"

	"github.com/maruel/circular"
	"go.uber.org/zap/zapcore"
)

const PtrSize = 32 << uintptr(^uintptr(0)>>63)

type Ring struct {
	zapcore.Core
	enc    zapcore.Encoder
	buffer circular.Buffer
}

func New(size uint) *Ring {
	return &Ring{buffer: circular.New(int(size))}
}

func (r *Ring) Close() {
	r.buffer.Flush() // ensures all readers have caught up.
	r.buffer.Close() // gracefully closes the readers.
}

func (r *Ring) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if r.Enabled(entry.Level) {
		return checked.AddCore(entry, r)
	}
	return checked
}

func (r *Ring) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// We need to be sure we're in 64-bit since circular buffer call LoadInt64
	// so if you're running in 32-bit the app will crash
	if PtrSize == 64 {
		buff, err := r.enc.EncodeEntry(entry, fields)
		if err != nil {
			return err
		}
		r.buffer.Write(buff.Bytes())
	}

	return r.Core.Write(entry, fields)
}

// WriteTo implements io.WriterTo
func (r *Ring) WriteTo(w io.Writer) (n int64, err error) {
	return r.buffer.WriteTo(w)
}

func (r *Ring) Wrap(core zapcore.Core, enc zapcore.Encoder) zapcore.Core {
	r.Core = core
	r.enc = enc
	return r
}
