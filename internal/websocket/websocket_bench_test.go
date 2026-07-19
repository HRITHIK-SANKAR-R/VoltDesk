package websocket

import (
	"bytes"
	"encoding/json"
	"sync"
	"testing"
)

var benchmarkPayload = []byte(`{"type":"chat_message","payload":{"content":"Hello World","conversation_id":"123","sender_id":"456"}}`)

type WsBenchEvent struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func BenchmarkStandardUnmarshal(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var incoming WsBenchEvent
			// Standard simulates allocating a new buffer per incoming read 
			tmp := make([]byte, len(benchmarkPayload))
			copy(tmp, benchmarkPayload)
			if err := json.Unmarshal(tmp, &incoming); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkPoolUnmarshal(b *testing.B) {
	// Re-using the bufferPool concept from client.go
	pool := sync.Pool{
		New: func() any { return new(bytes.Buffer) },
	}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// 1. Fetch from pool
			buf := pool.Get().(*bytes.Buffer)
			buf.Reset()
			
			// 2. Simulate reading from websocket io.Reader
			buf.Write(benchmarkPayload) 
			
			// 3. Unmarshal
			var incoming WsBenchEvent
			if err := json.Unmarshal(buf.Bytes(), &incoming); err != nil {
				b.Fatal(err)
			}
			
			// 4. Return to pool
			pool.Put(buf)
		}
	})
}
