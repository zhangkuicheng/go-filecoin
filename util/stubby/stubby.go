// Package stubby enables replacing methods or functions with "stub" implementations for tests.
package stubby

import (
	"os"
	"reflect"
	"strings"

	"github.com/filecoin-project/go-filecoin/util/chk"
)

// Registry manages a set of named stubs. The typical usage would be to make registry a member of some
// struct that needs stubs under test:
//
// type MyThing struct {
//     foo    string
//     bar    int
//     stubby stubby.Registry
// }
//
// func (mt MyThing) jump(int requestedHeight) (achievedHeight int) {
//     if f.stubby.Call("jump", requestedHeight, &achievedHeight) {
//         return
//     }
//	   ...
// }
//
// Then, under test:
//
// t := MyThing{}
// t.Add("jump", func(requestedHeight int) (achievedHeight int) {
//   return requestedHeight * 2
// })
// t.MethodThatCallsJumpInternally()
type Registry struct {
	m map[string]reflect.Value
}

var (
	underTest = strings.HasSuffix(os.Args[0], ".test")
)

// Add adds a named stub to the registry. The passed value must be a
// reference to a function.
func (s *Registry) Add(name string, fn interface{}) func() {
	if !underTest {
		panic("Add should not be called outside of tests")
	}

	if s.m == nil {
		s.m = map[string]reflect.Value{}
	}
	s.m[name] = reflect.ValueOf(fn)
	return func() {
		s.Remove(name)
	}
}

// Remove unregisters a stub.
func (s *Registry) Remove(name string) {
	if !underTest {
		panic("Remove should not be called outside of tests")
	}

	delete(s.m, name)
}

// Call invokes the named stub if there is one. Call should be invoked
// with all expected input arguments, plus pointers to all expected
// output arguments. The return function will set the output argument
// pointers before returning.
//
// If there is no matching stub, Call() returns false.
func (s Registry) Call(name string, args ...interface{}) (hadMatch bool) {
	if s.m == nil {
		return false
	}

	fn, ok := s.m[name]
	if !ok {
		return false
	}

	argValues := []reflect.Value{}

	// This is a little subtle because of variadic parameters. There
	// might be more or less argument than fn.Type().NumIn() and still
	// be a valid call. Luckily, Go doesn't support variadic *output*
	// parameters, so any set of inputs is still unambiguous.
	numIn := len(args) - fn.Type().NumOut()
	for i := 0; i < numIn; i++ {
		argValues = append(argValues, reflect.ValueOf(args[i]))
	}

	resultValues := fn.Call(argValues)

	for i, r := range resultValues {
		dstValue := reflect.ValueOf(args[numIn+i])
		chk.True(dstValue.Kind() == reflect.Ptr,
			"Invalid dst type for output arg: %v. Must be pointer.", dstValue.Type())
		reflect.Indirect(dstValue).Set(r)
	}

	return true
}
