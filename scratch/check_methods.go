package main
import (
	"fmt"
	"reflect"
	"github.com/2dprototype/wui"
)
func main() {
	var t *wui.StringTable
	typ := reflect.TypeOf(t)
	for i := 0; i < typ.NumMethod(); i++ {
		fmt.Println(typ.Method(i).Name)
	}
}
