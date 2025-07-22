// Copyright (c) nano Authors. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package component

import (
	"reflect"
	"unicode"
	"unicode/utf8"

	"github.com/lonng/nano/internal/utils/slices"
	"github.com/lonng/nano/session"
)

var (
	typeOfError   = reflect.TypeOf((*error)(nil)).Elem()
	typeOfBytes   = reflect.TypeOf(([]byte)(nil))
	typeOfSession = reflect.TypeOf(session.New(nil))
)

func isExported(name string) bool {
	w, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(w)
}

func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

// isHandlerMethod decide a method is suitable handler method
func isHandlerMethod(method reflect.Method) bool {
	//预留方法
	if slices.Contains(reservedMethods, method.Name) {
		return false
	}

	mt := method.Type
	// Method must be exported.
	if method.PkgPath != "" {
		return false
	}

	if mt.NumOut() > 2 {
		return false
	}

	return true
}

// resolveArgTypes 解析入参类型信息
func resolveArgTypes(mappingHandlerType reflect.Type) (argTypes []reflect.Type) {
	argsCount := mappingHandlerType.NumIn()
	argTypes = make([]reflect.Type, argsCount)
	for i := 0; i < argsCount; i++ {
		argTypes[i] = mappingHandlerType.In(i)
	}
	return
}

// resolveReturnTypes 解析返回值类型信息
func resolveReturnTypes(mappingHandlerType reflect.Type) (responseType reflect.Type, responseIndex int, errorIndex int) {
	responseType = nil
	responseIndex = -1
	errorIndex = -1
	for i, outCount := 0, mappingHandlerType.NumOut(); i < outCount; i++ {
		outType := mappingHandlerType.Out(i)
		if outType == typeOfError {
			errorIndex = i
		} else {
			responseType = outType
			responseIndex = i
		}
	}
	return
}
