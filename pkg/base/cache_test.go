package base

import (
	"fmt"
	"github.com/vmihailenco/msgpack/v5"
	"testing"
)

func TestMsgPack(t *testing.T) {
	type Item struct {
		Foo   string
		Slice []string
		Test  []Item
	}

	b, err := msgpack.Marshal(&Item{Foo: "bar", Slice: []string{"123", "456"}, Test: []Item{{Foo: "test"}}})
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))

	var item Item
	err = msgpack.Unmarshal(b, &item)
	if err != nil {
		panic(err)
	}
	fmt.Println(item)
}

func TestMsgPack2(t *testing.T) {
	type Item struct {
		Foo string
	}

	b, err := msgpack.Marshal(&[]Item{{Foo: "Bar"}, {Foo: "123"}})
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))

	var item []Item
	err = msgpack.Unmarshal(b, &item)
	if err != nil {
		panic(err)
	}
	fmt.Println(item)
}
