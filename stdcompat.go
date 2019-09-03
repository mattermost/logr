package logr

// StdLogger allows drop-in replacement of the stdlib log package.
// Use this when adding logr support to libraries so your users can chose
// to provide the stdlib logger or logr.
type StdLogger interface {
	Print(...interface{})
	Printf(string, ...interface{})
	Println(...interface{})

	Fatal(...interface{})
	Fatalf(string, ...interface{})
	Fatalln(...interface{})

	Panic(...interface{})
	Panicf(string, ...interface{})
	Panicln(...interface{})
}
