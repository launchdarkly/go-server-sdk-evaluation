package internal

import "testing"

var byteData = []byte(`abcdef`)

const stringData = "ghijklmno"

func BenchmarkLocalBufferWithLocallyPreallocatedSliceNoAlloc(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		localBuf := LocalBuffer{Data: make([]byte, 0, 100)}
		expectedLength := appendSomeValues(&localBuf)
		if len(localBuf.Data) != expectedLength {
			b.Fail()
		}
	}
}

func appendSomeValues(buf *LocalBuffer) int {
	expectedLength := 0
	buf.Append(byteData)
	expectedLength += len(byteData)
	buf.AppendString(stringData)
	expectedLength += len(stringData)
	buf.AppendByte('p')
	expectedLength += 1
	buf.AppendInt(999)
	expectedLength += 3
	return expectedLength
}
