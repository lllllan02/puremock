package main

import (
	"fmt"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/smartystreets/goconvey/convey"
)

func Test_mock(t *testing.T) {
	fmt.Println(mock("./test/local/local.go"))
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
