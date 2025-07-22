package assert

func Assert(guard bool, text string) {
	if !guard {
		panic(text)
	}
}
