package context

import (
	"context"

	"github.com/lllllan02/puremock/test/remote"
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
	ArrayType([]string, [10]string, []context.Context, []func(string))
	MapType(map[string]string, map[string][]string)
	SelectorExpr(context.Context, Local, remote.Remote, LocalInterface, remote.RemoteInterface, remote.Func)
	StarPointer(*int, context.Context)
	FuncType(func() error, func(string, string, string) (error, error), func(a, b, c string) (d, e, f string))
	InterfaceType(interface{}, []interface{})
	ChanType(chan int, chan []int, chan []func(string) []string) error
	Return() (int, string, []string, Local, LocalInterface, context.Context)
	Returns() (a, b, c int, d, e string)
	Funcs(a, b, c func()) (d, e, f func())
}
