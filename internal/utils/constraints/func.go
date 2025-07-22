package constraints

type (
	Action0                                                                                   func()
	Action1[T any]                                                                            func(T)
	Action2[T1 any, T2 any]                                                                   func(T1, T2)
	Action3[T1 any, T2 any, T3 any]                                                           func(T1, T2, T3)
	Action4[T1 any, T2 any, T3 any, T4 any]                                                   func(T1, T2, T3, T4)
	Action5[T1 any, T2 any, T3 any, T4 any, T5 any]                                           func(T1, T2, T3, T4, T5)
	Action6[T1 any, T2 any, T3 any, T4 any, T5 any, T6 any]                                   func(T1, T2, T3, T4, T5, T6)
	Action7[T1 any, T2 any, T3 any, T4 any, T5 any, T6 any, T7 any]                           func(T1, T2, T3, T4, T5, T6, T7)
	Action8[T1 any, T2 any, T3 any, T4 any, T5 any, T6 any, T7 any, T8 any]                   func(T1, T2, T3, T4, T5, T6, T7, T8)
	Action9[T1 any, T2 any, T3 any, T4 any, T5 any, T6 any, T7 any, T8 any, T9 any]           func(T1, T2, T3, T4, T5, T6, T7, T8, T9)
	Action10[T1 any, T2 any, T3 any, T4 any, T5 any, T6 any, T7 any, T8 any, T9 any, T10 any] func(T1, T2, T3, T4, T5, T6, T7, T8, T9, T10)

	Func0[TResult any]                                                                                   func() TResult
	Func1[T any, TResult any]                                                                            func(T) TResult
	Func2[T1 any, T2 any, TResult any]                                                                   func(T1, T2) TResult
	Func3[T1 any, T2 any, T3 any, TResult any]                                                           func(T1, T2, T3) TResult
	Func4[T1 any, T2 any, T3 any, T4 any, TResult any]                                                   func(T1, T2, T3, T4) TResult
	Func5[T1 any, T2 any, T3 any, T4 any, T5 any, TResult any]                                           func(T1, T2, T3, T4, T5) TResult
	Func6[T1 any, T2 any, T3 any, T4 any, T5 any, T6 any, TResult any]                                   func(T1, T2, T3, T4, T5, T6) TResult
	Func7[T1 any, T2 any, T3 any, T4 any, T5 any, T6 any, T7 any, TResult any]                           func(T1, T2, T3, T4, T5, T6, T7) TResult
	Func8[T1 any, T2 any, T3 any, T4 any, T5 any, T6 any, T7 any, T8 any, TResult any]                   func(T1, T2, T3, T4, T5, T6, T7, T8) TResult
	Func9[T1 any, T2 any, T3 any, T4 any, T5 any, T6 any, T7 any, T8 any, T9 any, TResult any]           func(T1, T2, T3, T4, T5, T6, T7, T8, T9) TResult
	Func10[T1 any, T2 any, T3 any, T4 any, T5 any, T6 any, T7 any, T8 any, T9 any, T10 any, TResult any] func(T1, T2, T3, T4, T5, T6, T7, T8, T9, T10) TResult

	Predicate0                                                                                   func() bool
	Predicate1[T any]                                                                            func(T) bool
	Predicate2[T1 any, T2 any]                                                                   func(T1, T2) bool
	Predicate3[T1 any, T2 any, T3 any]                                                           func(T1, T2, T3) bool
	Predicate4[T1 any, T2 any, T3 any, T4 any]                                                   func(T1, T2, T3, T4) bool
	Predicate5[T1 any, T2 any, T3 any, T4 any, T5 any]                                           func(T1, T2, T3, T4, T5) bool
	Predicate6[T1 any, T2 any, T3 any, T4 any, T5 any, T6 any]                                   func(T1, T2, T3, T4, T5, T6) bool
	Predicate7[T1 any, T2 any, T3 any, T4 any, T5 any, T6 any, T7 any]                           func(T1, T2, T3, T4, T5, T6, T7) bool
	Predicate8[T1 any, T2 any, T3 any, T4 any, T5 any, T6 any, T7 any, T8 any]                   func(T1, T2, T3, T4, T5, T6, T7, T8) bool
	Predicate9[T1 any, T2 any, T3 any, T4 any, T5 any, T6 any, T7 any, T8 any, T9 any]           func(T1, T2, T3, T4, T5, T6, T7, T8, T9) bool
	Predicate10[T1 any, T2 any, T3 any, T4 any, T5 any, T6 any, T7 any, T8 any, T9 any, T10 any] func(T1, T2, T3, T4, T5, T6, T7, T8, T9, T10) bool
)
