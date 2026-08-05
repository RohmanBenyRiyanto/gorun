package engine

import (
	"bufio"
	"os"
)

// stdinReader is shared by every prompt helper in this package instead of
// each one constructing its own bufio.NewReader(os.Stdin). A fresh reader
// per call works fine against a real terminal, where input only ever
// arrives one line at a time - but against a pipe (piped test input, or
// anything else feeding gorun's stdin non-interactively) the first read
// pulls the whole buffered chunk out of the OS pipe, and every reader
// created after that gets nothing back. One long-lived reader for the
// process's lifetime avoids losing whatever it over-read.
var stdinReader = bufio.NewReader(os.Stdin)
