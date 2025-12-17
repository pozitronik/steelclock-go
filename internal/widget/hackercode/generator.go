// Package hackercode provides a widget that displays procedurally generated code.
package hackercode

// CodeGenerator generates lines of procedural code.
type CodeGenerator interface {
	// NextLine returns the next line of generated code.
	NextLine() string

	// Reset resets the generator state for starting a new code block.
	Reset()
}
