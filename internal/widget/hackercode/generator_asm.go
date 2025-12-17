package hackercode

import (
	"fmt"
	"math/rand"
)

// Assembly registers and instructions.
var (
	asmRegisters32 = []string{
		"EAX", "EBX", "ECX", "EDX", "ESI", "EDI", "EBP", "ESP",
	}

	asmRegisters16 = []string{
		"AX", "BX", "CX", "DX", "SI", "DI", "BP", "SP",
	}

	asmRegisters8 = []string{
		"AL", "AH", "BL", "BH", "CL", "CH", "DL", "DH",
	}

	asmJumps = []string{
		"JMP", "JZ", "JNZ", "JE", "JNE", "JA", "JB", "JAE", "JBE",
		"JG", "JL", "JGE", "JLE", "JC", "JNC", "JS", "JNS",
	}

	asmComments = []string{
		"init registers",
		"check flags",
		"loop counter",
		"save context",
		"restore stack",
		"call handler",
		"return value",
		"clear buffer",
		"set pointer",
		"compare values",
	}
)

// AsmGenerator generates x86-style assembly code.
type AsmGenerator struct {
	rng       *rand.Rand
	lineCount int
	inProc    bool
}

// NewAsmGenerator creates a new assembly code generator.
func NewAsmGenerator(seed int64) *AsmGenerator {
	return &AsmGenerator{
		rng: rand.New(rand.NewSource(seed)),
	}
}

// NextLine generates the next line of assembly code.
func (g *AsmGenerator) NextLine() string {
	g.lineCount++

	// Occasionally generate a label
	if g.rng.Float64() < 0.08 {
		return g.generateLabel()
	}

	// Generate various instruction types
	instrType := g.rng.Intn(12)

	switch instrType {
	case 0, 1, 2:
		return g.generateDataMove()
	case 3, 4:
		return g.generateArithmetic()
	case 5:
		return g.generateJump()
	case 6:
		return g.generateStackOp()
	case 7:
		return g.generateCall()
	case 8:
		return g.generateMemoryOp()
	case 9:
		return g.generateComment()
	case 10:
		return g.generateCompare()
	default:
		return g.generateBitwise()
	}
}

// Reset resets the generator state.
func (g *AsmGenerator) Reset() {
	g.lineCount = 0
	g.inProc = false
}

// randomReg32 returns a random 32-bit register.
func (g *AsmGenerator) randomReg32() string {
	return asmRegisters32[g.rng.Intn(len(asmRegisters32))]
}

// randomReg16 returns a random 16-bit register.
func (g *AsmGenerator) randomReg16() string {
	return asmRegisters16[g.rng.Intn(len(asmRegisters16))]
}

// randomReg8 returns a random 8-bit register.
func (g *AsmGenerator) randomReg8() string {
	return asmRegisters8[g.rng.Intn(len(asmRegisters8))]
}

// randomReg returns a random register of any size.
func (g *AsmGenerator) randomReg() string {
	regType := g.rng.Intn(3)
	switch regType {
	case 0:
		return g.randomReg32()
	case 1:
		return g.randomReg16()
	default:
		return g.randomReg8()
	}
}

// randomHex8 returns a random 8-bit hex value.
func (g *AsmGenerator) randomHex8() string {
	return fmt.Sprintf("0x%02X", g.rng.Intn(256))
}

// randomHex16 returns a random 16-bit hex value.
func (g *AsmGenerator) randomHex16() string {
	return fmt.Sprintf("0x%04X", g.rng.Intn(65536))
}

// randomHex32 returns a random 32-bit hex value.
func (g *AsmGenerator) randomHex32() string {
	return fmt.Sprintf("0x%08X", g.rng.Uint32())
}

// randomAddr returns a random memory address.
func (g *AsmGenerator) randomAddr() string {
	// Generate address in typical code segment range
	base := uint32(0x00400000 + g.rng.Intn(0x10000))
	return fmt.Sprintf("0x%08X", base)
}

// randomMemRef returns a random memory reference.
func (g *AsmGenerator) randomMemRef() string {
	memType := g.rng.Intn(4)
	switch memType {
	case 0:
		return fmt.Sprintf("[%s]", g.randomReg32())
	case 1:
		return fmt.Sprintf("[%s+%s]", g.randomReg32(), g.randomHex8())
	case 2:
		return fmt.Sprintf("[%s]", g.randomAddr())
	default:
		return fmt.Sprintf("[%s+%s*4]", g.randomReg32(), g.randomReg32())
	}
}

// generateLabel generates a code label.
func (g *AsmGenerator) generateLabel() string {
	labelType := g.rng.Intn(3)
	switch labelType {
	case 0:
		return fmt.Sprintf("loc_%04X:", g.rng.Intn(65536))
	case 1:
		return fmt.Sprintf("sub_%04X:", g.rng.Intn(65536))
	default:
		return fmt.Sprintf("loop_%02X:", g.rng.Intn(256))
	}
}

// generateDataMove generates a MOV instruction.
func (g *AsmGenerator) generateDataMove() string {
	moveType := g.rng.Intn(4)
	switch moveType {
	case 0:
		return fmt.Sprintf("MOV %s, %s", g.randomReg32(), g.randomReg32())
	case 1:
		return fmt.Sprintf("MOV %s, %s", g.randomReg32(), g.randomHex32())
	case 2:
		return fmt.Sprintf("MOV %s, %s", g.randomReg32(), g.randomMemRef())
	default:
		return fmt.Sprintf("MOV %s, %s", g.randomMemRef(), g.randomReg32())
	}
}

// generateArithmetic generates arithmetic instructions.
func (g *AsmGenerator) generateArithmetic() string {
	ops := []string{"ADD", "SUB", "INC", "DEC", "NEG", "MUL", "IMUL"}
	op := ops[g.rng.Intn(len(ops))]

	if op == "INC" || op == "DEC" || op == "NEG" {
		return fmt.Sprintf("%s %s", op, g.randomReg32())
	}

	if op == "MUL" || op == "IMUL" {
		return fmt.Sprintf("%s %s", op, g.randomReg32())
	}

	operandType := g.rng.Intn(2)
	if operandType == 0 {
		return fmt.Sprintf("%s %s, %s", op, g.randomReg32(), g.randomReg32())
	}
	return fmt.Sprintf("%s %s, %s", op, g.randomReg32(), g.randomHex8())
}

// generateBitwise generates bitwise instructions.
func (g *AsmGenerator) generateBitwise() string {
	ops := []string{"XOR", "AND", "OR", "NOT", "SHL", "SHR", "ROL", "ROR"}
	op := ops[g.rng.Intn(len(ops))]

	if op == "NOT" {
		return fmt.Sprintf("%s %s", op, g.randomReg32())
	}

	if op == "SHL" || op == "SHR" || op == "ROL" || op == "ROR" {
		return fmt.Sprintf("%s %s, %d", op, g.randomReg32(), g.rng.Intn(8)+1)
	}

	operandType := g.rng.Intn(2)
	if operandType == 0 {
		return fmt.Sprintf("%s %s, %s", op, g.randomReg32(), g.randomReg32())
	}
	return fmt.Sprintf("%s %s, %s", op, g.randomReg32(), g.randomHex8())
}

// generateJump generates jump instructions.
func (g *AsmGenerator) generateJump() string {
	jump := asmJumps[g.rng.Intn(len(asmJumps))]

	targetType := g.rng.Intn(2)
	if targetType == 0 {
		return fmt.Sprintf("%s %s", jump, g.randomAddr())
	}
	return fmt.Sprintf("%s loc_%04X", jump, g.rng.Intn(65536))
}

// generateStackOp generates stack operations.
func (g *AsmGenerator) generateStackOp() string {
	op := []string{"PUSH", "POP"}[g.rng.Intn(2)]

	operandType := g.rng.Intn(2)
	if operandType == 0 {
		return fmt.Sprintf("%s %s", op, g.randomReg32())
	}
	if op == "PUSH" {
		return fmt.Sprintf("%s %s", op, g.randomHex32())
	}
	return fmt.Sprintf("%s %s", op, g.randomReg32())
}

// generateCall generates CALL/RET instructions.
func (g *AsmGenerator) generateCall() string {
	if g.rng.Float64() < 0.3 {
		return "RET"
	}

	targetType := g.rng.Intn(3)
	switch targetType {
	case 0:
		return fmt.Sprintf("CALL %s", g.randomAddr())
	case 1:
		return fmt.Sprintf("CALL sub_%04X", g.rng.Intn(65536))
	default:
		return fmt.Sprintf("CALL %s", g.randomReg32())
	}
}

// generateMemoryOp generates LEA and similar instructions.
func (g *AsmGenerator) generateMemoryOp() string {
	return fmt.Sprintf("LEA %s, %s", g.randomReg32(), g.randomMemRef())
}

// generateCompare generates CMP/TEST instructions.
func (g *AsmGenerator) generateCompare() string {
	op := []string{"CMP", "TEST"}[g.rng.Intn(2)]

	operandType := g.rng.Intn(2)
	if operandType == 0 {
		return fmt.Sprintf("%s %s, %s", op, g.randomReg32(), g.randomReg32())
	}
	return fmt.Sprintf("%s %s, %s", op, g.randomReg32(), g.randomHex8())
}

// generateComment generates an assembly comment.
func (g *AsmGenerator) generateComment() string {
	comment := asmComments[g.rng.Intn(len(asmComments))]
	return fmt.Sprintf("; %s", comment)
}
