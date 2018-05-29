// package stubby utilities for easily replacing methods or functions with "stub" implementations for tests.
package stubby

import (
	"reflect"
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
// func (mt MyThing) jump(int requestedHeight) (achivedHeight int) {
//     if f := f.stubby.Get("jump"); f != nil {
//         f(requestedHeight, &acheviedHeight)
//         return
//     }
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

// Add adds a named stub to the registry. The passed value must be a
// reference to a function.
func (s *Registry) Add(name string, fn interface{}) func() {
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
	delete(s.m, name)
}

// Get returns a wrapper for a stub that makes it convenient to execute
// from inside the real implementation. The returned function should be
// called with all expected input arguments, plus pointers to all expected
// output arguments. The return function will set the output argument
// pointers before returning.
//
// If there is no matching stub, Get() returns nil. Get() only allocates
// if there is in fact a registered stub, so there's no cost to use this
// in production code.
func (s Registry) Get(name string) func(args ...interface{}) {
	fn, ok := s.m[name]
	if ok {
		return func(args ...interface{}) {
			call(fn, args...)
		}
	}
	return nil
}

func call(fn reflect.Value, args ...interface{}) {
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
		if dstValue.Kind() != reflect.Ptr {
			panic("invalid dst type for output arg: " + dstValue.Kind().String())
		}
		reflect.Indirect(dstValue).Set(r)
	}
}
