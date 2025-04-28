package mjson

import (
	"go-im/api/access"
	"testing"

	"google.golang.org/protobuf/proto"
)

func BenchmarkMarshJson(b *testing.B) {
	a := access.Message{
		Type:  1,
		Data:  "faregsfaregsrgsrgs4235363565faregsrgsrgs4235363565faregsrgsrgs4235363565faregsrgsrgs4235363565rgsrgs4235363565",
		AckId: 0,
	}
	for b.Loop() {
		_, err := Marshal(&a)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkMarshProto(b *testing.B) {
	a := access.Message{
		Type:  1,
		Data:  "faregsfaregsrgsrgs4235363565faregsrgsrgs4235363565faregsrgsrgs4235363565faregsrgsrgs4235363565rgsrgs4235363565",
		AckId: 0,
	}
	for b.Loop() {
		_, err := proto.Marshal(&a)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkUnMarshJson(b *testing.B) {
	a := access.Message{
		Type:  1,
		Data:  "faregsfaregsrgsrgs4235363565faregsrgsrgs4235363565faregsrgsrgs4235363565faregsrgsrgs4235363565rgsrgs4235363565",
		AckId: 0,
	}
	c, err := Marshal(&a)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for b.Loop() {
		Unmarshal(c, &a)
	}
}

func BenchmarkUnMarshProto(b *testing.B) {
	a := access.Message{
		Type:  1,
		Data:  "faregsfaregsrgsrgs4235363565faregsrgsrgs4235363565faregsrgsrgs4235363565faregsrgsrgs4235363565rgsrgs4235363565",
		AckId: 0,
	}
	c, err := proto.Marshal(&a)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for b.Loop() {
		proto.Unmarshal(c, &a)
	}
}
