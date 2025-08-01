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

package mock_test

import (
	"testing"

	"github.com/lonng/nano/test/mock"
	"github.com/stretchr/testify/assert"
)

func TestNetworkEntity(t *testing.T) {
	entity := mock.NewNetworkEntity()

	assert.Nil(t, entity.LastResponse())
	assert.Equal(t, entity.LastMid(), uint64(1))
	assert.Nil(t, entity.Response("hello"))
	assert.Equal(t, entity.LastResponse().(string), "hello")

	assert.Nil(t, entity.FindResponseByMID(1))
	assert.Nil(t, entity.ResponseMid(1, "test"))
	assert.Equal(t, entity.FindResponseByMID(1).(string), "test")

	assert.Nil(t, entity.FindResponseByRoute("t.tt"))
	assert.Nil(t, entity.Push("t.tt", "test"))
	assert.Equal(t, entity.FindResponseByRoute("t.tt").(string), "test")

	assert.Equal(t, entity.RemoteAddr().String(), "mock-addr")
	assert.Nil(t, entity.Close())
}
