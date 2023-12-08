package main

import (
	"context"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/lllllan02/puremock/tmp"
	"github.com/smartystreets/goconvey/convey"
)

type Local struct{}
type LocalInterface interface{}

type Interface interface {
	Empty()
	Method(a, b, c string) (d, e, f string)
	IdentBool(bool)
	IdentNumber(int8, int16, int32, int64, int, uint8, uint16, uint32, uint64, uint, float32, float64, complex64, complex128)
	IdentString(rune, string)
	IdentError(error)
	Ellipsis(...string)
	ArrayType([]string, [10]string, []*context.Context, []func(string))
	MapType(map[string]string, map[string][]string)
	SelectorExpr(context.Context, Local, tmp.Remote, LocalInterface, tmp.RemoteInterface)
	StarPointer(*int, *context.Context)
	FuncType(func(), func(string, string, string) (error, error), func(a, b, c string) (d, e, f string))
	InterfaceType(interface{}, []interface{})
	ChanType(chan int, chan []int, chan []func(string) []string)
}

func Test_mock(t *testing.T) {
	mock("puremock_test.go")
}

func TestGetPackagePath(t *testing.T) {
	mockey.PatchConvey("1", t, func() {
		pkg, err := GetPackagePath("tmp/interface.go")
		convey.So(pkg, convey.ShouldEqual, "github.com/lllllan02/puremock/tmp")
		convey.So(err, convey.ShouldEqual, nil)
	})

	mockey.PatchConvey("2", t, func() {
		_, err := GetPackagePath("../")
		convey.So(err, convey.ShouldBeError)
	})
}
