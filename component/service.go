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
	"errors"
	"reflect"

	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/nap"
	"github.com/lonng/nano/serialize/protobuf"
)

// Service implements a specific service, some of it's methods will be
// called when the correspond events is occurred.
type Service struct {
	Name      string              // name of service
	Type      reflect.Type        // type of the receiver
	Receiver  reflect.Value       // receiver of methods for the service
	Handlers  map[string]*Handler // registered methods
	SchedName string              // name of scheduler variable in session data
	Options   options             // options
}

func NewService(comp Component, opts []Option) *Service {
	s := &Service{
		Type:     reflect.TypeOf(comp),
		Receiver: reflect.ValueOf(comp),
	}

	// apply options
	for i := range opts {
		opt := opts[i]
		opt(&s.Options)
	}
	if name := s.Options.name; name != "" {
		s.Name = name
	} else {
		s.Name = reflect.Indirect(s.Receiver).Type().Name()
	}
	s.SchedName = s.Options.schedName

	return s
}

// ExtractHandler extract the set of methods from the
// receiver value which satisfy the following conditions:
// - exported method of exported type
// - two arguments, both of exported type
// - the first argument is *session.Session
// - the second argument is []byte or a pointer
func (s *Service) ExtractHandler() error {
	typeName := reflect.Indirect(s.Receiver).Type().Name()
	if typeName == "" {
		return errors.New("no service name for type " + s.Type.String())
	}
	if !isExported(typeName) {
		return errors.New("type " + typeName + " is not exported")
	}

	// Install the methods
	s.Handlers = s.suitableHandlerMethods(s.Receiver, s.Type)

	if len(s.Handlers) == 0 {
		// To help the user, see if a pointer receiver would work.
		method := s.suitableHandlerMethods(s.Receiver, reflect.PtrTo(s.Type))
		if len(method) == 0 {
			return errors.New("type " + s.Name + " has no exported methods of suitable type")
		}
		return errors.New("type " + s.Name + " has no exported methods of suitable type (hint: pass a pointer to value of that type)")
	}

	return nil
}

// suitableMethods returns suitable methods of typ
func (s *Service) suitableHandlerMethods(receiver reflect.Value, typ reflect.Type) map[string]*Handler {
	methods := make(map[string]*Handler)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mn := method.Name
		if isHandlerMethod(method) {
			// rewrite handler name
			if s.Options.nameFunc != nil {
				mn = s.Options.nameFunc(mn)
			}
			methods[mn] = NewHandlerFromMethod(receiver, method)
		}
	}
	return methods
}

// Handler 切点信息
type Handler struct {
	IsMethod bool
	Receiver reflect.Value // receiver of method
	//
	handlerType   reflect.Type
	handlerValue  reflect.Value
	argTypes      []reflect.Type
	responseType  reflect.Type
	responseIndex int
	errorIndex    int
}

// NewHandlerFromMethod 从方法创建
func NewHandlerFromMethod(receiver reflect.Value, method reflect.Method) *Handler {
	return newHandler(receiver, method.Func)
}

// NewHandlerFromFunc 从函数创建
func NewHandlerFromFunc(handler any) *Handler {
	return newHandler(reflect.Value{}, reflect.ValueOf(handler))
}

// newHandler 创建一个处理函数的切点
func newHandler(receiver, handlerValue reflect.Value) *Handler {
	//处理函数的反射类型
	handlerType := handlerValue.Type()
	if handlerType.Kind() != reflect.Func {
		panic("handler [" + handlerType.String() + "] 必须是函数")
	}
	if handlerType.NumOut() > 2 {
		panic("handler [" + handlerType.String() + "] 返回值不能超过 2 个")
	}
	//解析入参信息
	argTypes := resolveArgTypes(handlerType)
	//解析返回值信息
	responseType, responseIndex, errorIndex := resolveReturnTypes(handlerType)
	//定义切点, 织入处理函数, 后续可追加处理函数; 此处预先埋入系统级拦截器
	return &Handler{
		IsMethod: receiver.IsValid(),
		Receiver: receiver,
		//
		handlerType:   handlerType,
		handlerValue:  handlerValue,
		argTypes:      argTypes,
		responseType:  responseType,
		responseIndex: responseIndex,
		errorIndex:    errorIndex,
	}
}

// Call 执行处理函数
func (h *Handler) Call(c *nap.Context) {
	args, err := h.ResolveArgs(c)
	if err != nil {
		c.Error(err)
		log.Info("handler call resolve args error.", err)
		return
	}
	retValues := h.handlerValue.Call(args)
	response, err := h.ResolveReturnValues(retValues)
	if err != nil {
		c.Error(err)
		log.Info("handler call resolve return values error.", err)
		return
	}
	if response == nil {
		return
	}
	err = c.Response(response)
	if err != nil {
		c.Error(err)
		log.Info("handler call response error.", err)
	}
}

// ResolveArgs 解析入参
func (h *Handler) ResolveArgs(c *nap.Context) ([]reflect.Value, error) {
	args := make([]reflect.Value, len(h.argTypes))
	for i, argType := range h.argTypes {
		arg, err := h.ResolveArg(c, i, argType)
		if err != nil {
			return nil, err
		}
		args[i] = arg
	}
	return args, nil
}

// ResolveArg 解析单个入参
func (h *Handler) ResolveArg(c *nap.Context, i int, argType reflect.Type) (reflect.Value, error) {
	// 方法的接收者
	if i == 0 && h.IsMethod {
		return h.Receiver, nil
	}
	// Context 类型
	if argType == nap.ContextType {
		return reflect.ValueOf(c), nil
	}
	// *Session 类型
	if argType == typeOfSession {
		return reflect.ValueOf(c.Session), nil
	}
	// []byte 类型
	if argType == typeOfBytes {
		return reflect.ValueOf(c.Msg.Data), nil
	}
	// 其他类型 json: 结构体, 指针; protobuf: 结构体
	ptrToArg := reflect.New(argType) //ptrToArg 即指向 arg 的指针, 不论 arg 为值类型还是指针类型
	ptr := ptrToArg.Interface()
	err := c.ShouldBind(ptr)
	if err == nil {
		return ptrToArg.Elem(), nil
	}
	// 其他类型 protobuf: 指针
	if errors.Is(err, protobuf.ErrWrongValueType) && argType.Kind() == reflect.Ptr {
		ptrOfArg := reflect.New(argType.Elem()) //ptrOfArg 即 arg 自身为指针
		ptr = ptrOfArg.Interface()
		err = c.ShouldBind(ptr)
		if err != nil {
			return reflect.Value{}, err
		}
		return ptrOfArg, nil
	}
	return reflect.Value{}, err
}

// ResolveReturnValues 解析返回值
func (h *Handler) ResolveReturnValues(retValues []reflect.Value) (response any, err error) {
	if h.responseIndex >= 0 {
		response = retValues[h.responseIndex].Interface()
	}
	if h.errorIndex >= 0 {
		errObj := retValues[h.errorIndex].Interface()
		if errObj != nil {
			err = errObj.(error)
		}
	}
	return
}
