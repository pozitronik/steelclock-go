package hackercode

import (
	"fmt"
	"math/rand"
	"strings"
)

// Variable and type names for C-like code generation.
var (
	cVarNames = []string{
		"ptr", "buf", "data", "idx", "cnt", "tmp", "node", "head", "next",
		"status", "flags", "result", "len", "size", "offset", "val", "ret",
		"src", "dst", "key", "hash", "mask", "base", "addr", "ctx", "cfg",
	}

	cTypeNames = []string{
		"void", "int", "char", "uint8_t", "uint16_t", "uint32_t", "size_t",
		"int8_t", "int16_t", "int32_t", "uintptr_t", "bool",
	}

	cStructNames = []string{
		"node_t", "ctx_t", "cfg_t", "buf_t", "msg_t", "hdr_t", "entry_t",
	}

	cFuncNames = []string{
		"init", "process", "handle", "parse", "validate", "encode", "decode",
		"alloc", "free", "read", "write", "send", "recv", "check", "update",
		"get", "set", "find", "insert", "remove", "clear", "reset", "sync",
	}

	cComments = []string{
		"initialize buffer",
		"check status",
		"validate input",
		"handle error",
		"process data",
		"update state",
		"cleanup resources",
		"parse header",
		"encode payload",
		"sync state",
	}
)

// CGenerator generates C-like code.
type CGenerator struct {
	rng         *rand.Rand
	inFunction  bool
	indentLevel int
	lineCount   int
	inBlock     bool // inside if/for/while block
	blockDepth  int
}

// NewCGenerator creates a new C-like code generator.
func NewCGenerator(seed int64) *CGenerator {
	return &CGenerator{
		rng: rand.New(rand.NewSource(seed)),
	}
}

// NextLine generates the next line of C-like code.
func (g *CGenerator) NextLine() string {
	g.lineCount++

	// Decide what to generate based on context
	if !g.inFunction && g.rng.Float64() < 0.25 {
		return g.generateFunctionHeader()
	}

	if g.inFunction {
		// Maybe end the function
		if g.lineCount > 6 && g.indentLevel == 1 && g.rng.Float64() < 0.15 {
			return g.generateFunctionEnd()
		}

		// Maybe end a block
		if g.inBlock && g.blockDepth > 0 && g.rng.Float64() < 0.2 {
			return g.generateBlockEnd()
		}

		// Maybe start a block
		if g.indentLevel < 3 && g.rng.Float64() < 0.15 {
			return g.generateBlockStart()
		}
	}

	return g.generateStatement()
}

// Reset resets the generator state.
func (g *CGenerator) Reset() {
	g.inFunction = false
	g.indentLevel = 0
	g.lineCount = 0
	g.inBlock = false
	g.blockDepth = 0
}

// indent returns the current indentation string.
func (g *CGenerator) indent() string {
	return strings.Repeat("  ", g.indentLevel)
}

// randomVar returns a random variable name.
func (g *CGenerator) randomVar() string {
	return cVarNames[g.rng.Intn(len(cVarNames))]
}

// randomType returns a random type name.
func (g *CGenerator) randomType() string {
	return cTypeNames[g.rng.Intn(len(cTypeNames))]
}

// randomStruct returns a random struct name.
func (g *CGenerator) randomStruct() string {
	return cStructNames[g.rng.Intn(len(cStructNames))]
}

// randomFunc returns a random function name.
func (g *CGenerator) randomFunc() string {
	return cFuncNames[g.rng.Intn(len(cFuncNames))]
}

// randomHex8 returns a random 8-bit hex value.
func (g *CGenerator) randomHex8() string {
	return fmt.Sprintf("0x%02X", g.rng.Intn(256))
}

// randomHex16 returns a random 16-bit hex value.
func (g *CGenerator) randomHex16() string {
	return fmt.Sprintf("0x%04X", g.rng.Intn(65536))
}

// randomHex32 returns a random 32-bit hex value.
func (g *CGenerator) randomHex32() string {
	return fmt.Sprintf("0x%08X", g.rng.Uint32())
}

// randomInt returns a random small integer.
func (g *CGenerator) randomInt() int {
	return g.rng.Intn(256)
}

// generateFunctionHeader generates a function definition header.
func (g *CGenerator) generateFunctionHeader() string {
	g.inFunction = true
	g.indentLevel = 1
	g.lineCount = 0

	returnType := g.randomType()
	funcName := g.randomFunc()
	suffix := fmt.Sprintf("_%s", g.randomVar())

	// Generate parameters
	paramCount := g.rng.Intn(3)
	var params []string
	for i := 0; i < paramCount; i++ {
		paramType := g.randomType()
		paramName := g.randomVar()
		if g.rng.Float64() < 0.3 {
			params = append(params, fmt.Sprintf("%s *%s", paramType, paramName))
		} else {
			params = append(params, fmt.Sprintf("%s %s", paramType, paramName))
		}
	}

	paramStr := strings.Join(params, ", ")
	if paramStr == "" {
		paramStr = "void"
	}

	return fmt.Sprintf("%s %s%s(%s) {", returnType, funcName, suffix, paramStr)
}

// generateFunctionEnd generates a function closing brace.
func (g *CGenerator) generateFunctionEnd() string {
	g.inFunction = false
	g.indentLevel = 0
	g.inBlock = false
	g.blockDepth = 0
	return "}"
}

// generateBlockStart generates the start of an if/for/while block.
func (g *CGenerator) generateBlockStart() string {
	g.inBlock = true
	g.blockDepth++

	blockType := g.rng.Intn(3)
	indent := g.indent()
	g.indentLevel++

	switch blockType {
	case 0: // if statement
		condition := g.generateCondition()
		return fmt.Sprintf("%sif (%s) {", indent, condition)
	case 1: // for loop
		limit := g.randomInt()
		if limit < 8 {
			limit = 8
		}
		return fmt.Sprintf("%sfor (int i = 0; i < %d; i++) {", indent, limit)
	default: // while loop
		condition := g.generateCondition()
		return fmt.Sprintf("%swhile (%s) {", indent, condition)
	}
}

// generateBlockEnd generates a closing brace for a block.
func (g *CGenerator) generateBlockEnd() string {
	if g.indentLevel > 1 {
		g.indentLevel--
	}
	g.blockDepth--
	if g.blockDepth <= 0 {
		g.inBlock = false
		g.blockDepth = 0
	}
	return fmt.Sprintf("%s}", g.indent())
}

// generateCondition generates a conditional expression.
func (g *CGenerator) generateCondition() string {
	condType := g.rng.Intn(4)
	v := g.randomVar()

	switch condType {
	case 0:
		return fmt.Sprintf("%s & %s", v, g.randomHex8())
	case 1:
		return fmt.Sprintf("%s == %s", v, g.randomHex8())
	case 2:
		return fmt.Sprintf("%s != NULL", v)
	default:
		return fmt.Sprintf("%s < %d", v, g.randomInt())
	}
}

// generateStatement generates a single statement.
func (g *CGenerator) generateStatement() string {
	indent := g.indent()

	stmtType := g.rng.Intn(12)

	switch stmtType {
	case 0: // Variable declaration with hex init
		return fmt.Sprintf("%s%s %s = %s;", indent, g.randomType(), g.randomVar(), g.randomHex16())

	case 1: // Variable declaration with int init
		return fmt.Sprintf("%s%s %s = %d;", indent, g.randomType(), g.randomVar(), g.randomInt())

	case 2: // Pointer assignment
		return fmt.Sprintf("%s*%s = %s;", indent, g.randomVar(), g.randomHex8())

	case 3: // Struct member access
		return fmt.Sprintf("%s%s->%s = %s;", indent, g.randomVar(), g.randomVar(), g.randomHex16())

	case 4: // Function call
		return fmt.Sprintf("%s%s_%s(%s);", indent, g.randomFunc(), g.randomVar(), g.randomVar())

	case 5: // Return statement
		if g.inFunction && g.rng.Float64() < 0.5 {
			return fmt.Sprintf("%sreturn %s;", indent, g.randomHex8())
		}
		return fmt.Sprintf("%sreturn %s;", indent, g.randomVar())

	case 6: // Bitwise operation
		op := []string{"&", "|", "^", ">>", "<<"}[g.rng.Intn(5)]
		return fmt.Sprintf("%s%s %s= %s;", indent, g.randomVar(), op, g.randomHex8())

	case 7: // Array access
		return fmt.Sprintf("%s%s[%d] = %s;", indent, g.randomVar(), g.rng.Intn(64), g.randomHex8())

	case 8: // Comment
		comment := cComments[g.rng.Intn(len(cComments))]
		return fmt.Sprintf("%s// %s", indent, comment)

	case 9: // Increment/decrement
		op := []string{"++", "--"}[g.rng.Intn(2)]
		return fmt.Sprintf("%s%s%s;", indent, g.randomVar(), op)

	case 10: // Cast assignment
		return fmt.Sprintf("%s%s = (%s)%s;", indent, g.randomVar(), g.randomType(), g.randomVar())

	default: // Simple assignment
		return fmt.Sprintf("%s%s = %s;", indent, g.randomVar(), g.randomVar())
	}
}
