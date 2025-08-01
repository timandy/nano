// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slices

import (
	"math"
	"strings"
	"testing"

	"github.com/lonng/nano/internal/utils/cmp"
)

var equalIntTests = []struct {
	s1, s2 []int
	want   bool
}{
	{
		[]int{1},
		nil,
		false,
	},
	{
		[]int{},
		nil,
		true,
	},
	{
		[]int{1, 2, 3},
		[]int{1, 2, 3},
		true,
	},
	{
		[]int{1, 2, 3},
		[]int{1, 2, 3, 4},
		false,
	},
}

var equalFloatTests = []struct {
	s1, s2       []float64
	wantEqual    bool
	wantEqualNaN bool
}{
	{
		[]float64{1, 2},
		[]float64{1, 2},
		true,
		true,
	},
	{
		[]float64{1, 2, math.NaN()},
		[]float64{1, 2, math.NaN()},
		false,
		true,
	},
}

func TestEqual(t *testing.T) {
	for _, test := range equalIntTests {
		if got := Equal(test.s1, test.s2); got != test.want {
			t.Errorf("Equal(%v, %v) = %t, want %t", test.s1, test.s2, got, test.want)
		}
	}
	for _, test := range equalFloatTests {
		if got := Equal(test.s1, test.s2); got != test.wantEqual {
			t.Errorf("Equal(%v, %v) = %t, want %t", test.s1, test.s2, got, test.wantEqual)
		}
	}
}

// equal is simply ==.
func equal[T comparable](v1, v2 T) bool {
	return v1 == v2
}

// equalNaN is like == except that all NaNs are equal.
func equalNaN[T comparable](v1, v2 T) bool {
	isNaN := func(f T) bool { return f != f }
	return v1 == v2 || (isNaN(v1) && isNaN(v2))
}

// offByOne returns true if integers v1 and v2 differ by 1.
func offByOne(v1, v2 int) bool {
	return v1 == v2+1 || v1 == v2-1
}

func TestEqualFunc(t *testing.T) {
	for _, test := range equalIntTests {
		if got := EqualFunc(test.s1, test.s2, equal[int]); got != test.want {
			t.Errorf("EqualFunc(%v, %v, equal[int]) = %t, want %t", test.s1, test.s2, got, test.want)
		}
	}
	for _, test := range equalFloatTests {
		if got := EqualFunc(test.s1, test.s2, equal[float64]); got != test.wantEqual {
			t.Errorf("Equal(%v, %v, equal[float64]) = %t, want %t", test.s1, test.s2, got, test.wantEqual)
		}
		if got := EqualFunc(test.s1, test.s2, equalNaN[float64]); got != test.wantEqualNaN {
			t.Errorf("Equal(%v, %v, equalNaN[float64]) = %t, want %t", test.s1, test.s2, got, test.wantEqualNaN)
		}
	}

	s1 := []int{1, 2, 3}
	s2 := []int{2, 3, 4}
	if EqualFunc(s1, s1, offByOne) {
		t.Errorf("EqualFunc(%v, %v, offByOne) = true, want false", s1, s1)
	}
	if !EqualFunc(s1, s2, offByOne) {
		t.Errorf("EqualFunc(%v, %v, offByOne) = false, want true", s1, s2)
	}

	s3 := []string{"a", "b", "c"}
	s4 := []string{"A", "B", "C"}
	if !EqualFunc(s3, s4, strings.EqualFold) {
		t.Errorf("EqualFunc(%v, %v, strings.EqualFold) = false, want true", s3, s4)
	}

	cmpIntString := func(v1 int, v2 string) bool {
		return string(rune(v1)-1+'a') == v2
	}
	if !EqualFunc(s1, s3, cmpIntString) {
		t.Errorf("EqualFunc(%v, %v, cmpIntString) = false, want true", s1, s3)
	}
}

func BenchmarkEqualFunc_Large(b *testing.B) {
	type Large [4 * 1024]byte

	xs := make([]Large, 1024)
	ys := make([]Large, 1024)
	for i := 0; i < b.N; i++ {
		_ = EqualFunc(xs, ys, func(x, y Large) bool { return x == y })
	}
}

var compareIntTests = []struct {
	s1, s2 []int
	want   int
}{
	{
		[]int{1},
		[]int{1},
		0,
	},
	{
		[]int{1},
		[]int{},
		1,
	},
	{
		[]int{},
		[]int{1},
		-1,
	},
	{
		[]int{},
		[]int{},
		0,
	},
	{
		[]int{1, 2, 3},
		[]int{1, 2, 3},
		0,
	},
	{
		[]int{1, 2, 3},
		[]int{1, 2, 3, 4},
		-1,
	},
	{
		[]int{1, 2, 3, 4},
		[]int{1, 2, 3},
		+1,
	},
	{
		[]int{1, 2, 3},
		[]int{1, 4, 3},
		-1,
	},
	{
		[]int{1, 4, 3},
		[]int{1, 2, 3},
		+1,
	},
	{
		[]int{1, 4, 3},
		[]int{1, 2, 3, 8, 9},
		+1,
	},
}

var compareFloatTests = []struct {
	s1, s2 []float64
	want   int
}{
	{
		[]float64{},
		[]float64{},
		0,
	},
	{
		[]float64{1},
		[]float64{1},
		0,
	},
	{
		[]float64{math.NaN()},
		[]float64{math.NaN()},
		0,
	},
	{
		[]float64{1, 2, math.NaN()},
		[]float64{1, 2, math.NaN()},
		0,
	},
	{
		[]float64{1, math.NaN(), 3},
		[]float64{1, math.NaN(), 4},
		-1,
	},
	{
		[]float64{1, math.NaN(), 3},
		[]float64{1, 2, 4},
		-1,
	},
	{
		[]float64{1, math.NaN(), 3},
		[]float64{1, 2, math.NaN()},
		-1,
	},
	{
		[]float64{1, 2, 3},
		[]float64{1, 2, math.NaN()},
		+1,
	},
	{
		[]float64{1, 2, 3},
		[]float64{1, math.NaN(), 3},
		+1,
	},
	{
		[]float64{1, math.NaN(), 3, 4},
		[]float64{1, 2, math.NaN()},
		-1,
	},
}

func TestCompare(t *testing.T) {
	intWant := func(want bool) string {
		if want {
			return "0"
		}
		return "!= 0"
	}
	for _, test := range equalIntTests {
		if got := Compare(test.s1, test.s2); (got == 0) != test.want {
			t.Errorf("Compare(%v, %v) = %d, want %s", test.s1, test.s2, got, intWant(test.want))
		}
	}
	for _, test := range equalFloatTests {
		if got := Compare(test.s1, test.s2); (got == 0) != test.wantEqualNaN {
			t.Errorf("Compare(%v, %v) = %d, want %s", test.s1, test.s2, got, intWant(test.wantEqualNaN))
		}
	}

	for _, test := range compareIntTests {
		if got := Compare(test.s1, test.s2); got != test.want {
			t.Errorf("Compare(%v, %v) = %d, want %d", test.s1, test.s2, got, test.want)
		}
	}
	for _, test := range compareFloatTests {
		if got := Compare(test.s1, test.s2); got != test.want {
			t.Errorf("Compare(%v, %v) = %d, want %d", test.s1, test.s2, got, test.want)
		}
	}
}

func equalToCmp[T comparable](eq func(T, T) bool) func(T, T) int {
	return func(v1, v2 T) int {
		if eq(v1, v2) {
			return 0
		}
		return 1
	}
}

func TestCompareFunc(t *testing.T) {
	intWant := func(want bool) string {
		if want {
			return "0"
		}
		return "!= 0"
	}
	for _, test := range equalIntTests {
		if got := CompareFunc(test.s1, test.s2, equalToCmp(equal[int])); (got == 0) != test.want {
			t.Errorf("CompareFunc(%v, %v, equalToCmp(equal[int])) = %d, want %s", test.s1, test.s2, got, intWant(test.want))
		}
	}
	for _, test := range equalFloatTests {
		if got := CompareFunc(test.s1, test.s2, equalToCmp(equal[float64])); (got == 0) != test.wantEqual {
			t.Errorf("CompareFunc(%v, %v, equalToCmp(equal[float64])) = %d, want %s", test.s1, test.s2, got, intWant(test.wantEqual))
		}
	}

	for _, test := range compareIntTests {
		if got := CompareFunc(test.s1, test.s2, cmp.Compare[int]); got != test.want {
			t.Errorf("CompareFunc(%v, %v, cmp[int]) = %d, want %d", test.s1, test.s2, got, test.want)
		}
	}
	for _, test := range compareFloatTests {
		if got := CompareFunc(test.s1, test.s2, cmp.Compare[float64]); got != test.want {
			t.Errorf("CompareFunc(%v, %v, cmp[float64]) = %d, want %d", test.s1, test.s2, got, test.want)
		}
	}

	s1 := []int{1, 2, 3}
	s2 := []int{2, 3, 4}
	if got := CompareFunc(s1, s2, equalToCmp(offByOne)); got != 0 {
		t.Errorf("CompareFunc(%v, %v, offByOne) = %d, want 0", s1, s2, got)
	}

	s3 := []string{"a", "b", "c"}
	s4 := []string{"A", "B", "C"}
	if got := CompareFunc(s3, s4, strings.Compare); got != 1 {
		t.Errorf("CompareFunc(%v, %v, strings.Compare) = %d, want 1", s3, s4, got)
	}

	compareLower := func(v1, v2 string) int {
		return strings.Compare(strings.ToLower(v1), strings.ToLower(v2))
	}
	if got := CompareFunc(s3, s4, compareLower); got != 0 {
		t.Errorf("CompareFunc(%v, %v, compareLower) = %d, want 0", s3, s4, got)
	}

	cmpIntString := func(v1 int, v2 string) int {
		return strings.Compare(string(rune(v1)-1+'a'), v2)
	}
	if got := CompareFunc(s1, s3, cmpIntString); got != 0 {
		t.Errorf("CompareFunc(%v, %v, cmpIntString) = %d, want 0", s1, s3, got)
	}
}

var indexTests = []struct {
	s    []int
	v    int
	want int
}{
	{
		nil,
		0,
		-1,
	},
	{
		[]int{},
		0,
		-1,
	},
	{
		[]int{1, 2, 3},
		2,
		1,
	},
	{
		[]int{1, 2, 2, 3},
		2,
		1,
	},
	{
		[]int{1, 2, 3, 2},
		2,
		1,
	},
}

func TestIndex(t *testing.T) {
	for _, test := range indexTests {
		if got := Index(test.s, test.v); got != test.want {
			t.Errorf("Index(%v, %v) = %d, want %d", test.s, test.v, got, test.want)
		}
	}
}

func equalToIndex[T any](f func(T, T) bool, v1 T) func(T) bool {
	return func(v2 T) bool {
		return f(v1, v2)
	}
}

func BenchmarkIndex_Large(b *testing.B) {
	type Large [4 * 1024]byte

	ss := make([]Large, 1024)
	for i := 0; i < b.N; i++ {
		_ = Index(ss, Large{1})
	}
}

func TestIndexFunc(t *testing.T) {
	for _, test := range indexTests {
		if got := IndexFunc(test.s, equalToIndex(equal[int], test.v)); got != test.want {
			t.Errorf("IndexFunc(%v, equalToIndex(equal[int], %v)) = %d, want %d", test.s, test.v, got, test.want)
		}
	}

	s1 := []string{"hi", "HI"}
	if got := IndexFunc(s1, equalToIndex(equal[string], "HI")); got != 1 {
		t.Errorf("IndexFunc(%v, equalToIndex(equal[string], %q)) = %d, want %d", s1, "HI", got, 1)
	}
	if got := IndexFunc(s1, equalToIndex(strings.EqualFold, "HI")); got != 0 {
		t.Errorf("IndexFunc(%v, equalToIndex(strings.EqualFold, %q)) = %d, want %d", s1, "HI", got, 0)
	}
}

func BenchmarkIndexFunc_Large(b *testing.B) {
	type Large [4 * 1024]byte

	ss := make([]Large, 1024)
	for i := 0; i < b.N; i++ {
		_ = IndexFunc(ss, func(e Large) bool {
			return e == Large{1}
		})
	}
}

func TestContains(t *testing.T) {
	for _, test := range indexTests {
		if got := Contains(test.s, test.v); got != (test.want != -1) {
			t.Errorf("Contains(%v, %v) = %t, want %t", test.s, test.v, got, test.want != -1)
		}
	}
}

func TestContainsFunc(t *testing.T) {
	for _, test := range indexTests {
		if got := ContainsFunc(test.s, equalToIndex(equal[int], test.v)); got != (test.want != -1) {
			t.Errorf("ContainsFunc(%v, equalToIndex(equal[int], %v)) = %t, want %t", test.s, test.v, got, test.want != -1)
		}
	}

	s1 := []string{"hi", "HI"}
	if got := ContainsFunc(s1, equalToIndex(equal[string], "HI")); got != true {
		t.Errorf("ContainsFunc(%v, equalToContains(equal[string], %q)) = %t, want %t", s1, "HI", got, true)
	}
	if got := ContainsFunc(s1, equalToIndex(equal[string], "hI")); got != false {
		t.Errorf("ContainsFunc(%v, equalToContains(strings.EqualFold, %q)) = %t, want %t", s1, "hI", got, false)
	}
	if got := ContainsFunc(s1, equalToIndex(strings.EqualFold, "hI")); got != true {
		t.Errorf("ContainsFunc(%v, equalToContains(strings.EqualFold, %q)) = %t, want %t", s1, "hI", got, true)
	}
}
