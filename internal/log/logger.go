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

package log

import (
	"log"
	"os"
)

// Logger represents  the log interface
type Logger interface {
	Println(v ...any)
	Fatal(v ...any)
	Fatalf(format string, v ...any)
}

func init() {
	SetLogger(log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile))
}

var (
	Println func(v ...any)
	Fatal   func(v ...any)
	Fatalf  func(format string, v ...any)
)

// SetLogger rewrites the default logger
func SetLogger(logger Logger) {
	if logger == nil {
		return
	}
	Println = logger.Println
	Fatal = logger.Fatal
	Fatalf = logger.Fatalf
}
