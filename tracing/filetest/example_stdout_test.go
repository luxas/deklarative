package filetest

func ExampleExampleStdout() {
	_, _ = ExampleStdout.Write([]byte(`
hello   
		much whitespace before and after this line indeed 	
   	    
`))

	// Output:
	// hello
	// 		much whitespace before and after this line indeed
}
