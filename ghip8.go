/*

	Infro from
	http://devernay.free.fr/hacks/chip8/C8TECH10.HTM#1.0

*/

package ghip8

import (
	"errors"
	"fmt"
	"math/rand"
	"time"
)

type Chip8 struct {
	// Internal memory
	memory [4096]uint8
	// V0 - VF registers
	Register [16]uint8
	// I register
	I uint16
	// Sound and delay counters
	Delay, Sound uint8

	// Program counter
	pc uint16
	// Stack pointer
	stack *stack
	// 64x32 pixel video memory
	VideoMemory [256]uint8

	// Program lenght
	prgEnd uint16

	// keyboard
	keyboard [16]uint8
}

// Stack is a basic LIFO stack that resizes as needed.
type stack struct {
	nodes []uint16
	count int
}

// Push adds a node to the stack.
func (s *stack) push(n uint16) {
	s.nodes = append(s.nodes[:s.count], n)
	s.count++
}

// Pop removes and returns a node from the stack in last to first order.
func (s *stack) pop() uint16 {
	if s.count == 0 {
		return 0
	}
	s.count--
	return s.nodes[s.count]
}

// Instruction params
type InstructionParm struct {
	//address 16bit
	Addr uint16
	// byte
	Byte uint8
	// reg 1
	Vx uint8
	// reg 2
	Vy uint8
}

// Instruction
type Instruction struct {
	Opcode uint16
	// bitmask for opcode recognition
	// FFFF -> exact match
	// F000 -> match first 4 bits
	// F00F -> match first and last 4 bits
	// and so on
	Bitmask uint16
	// format used decompiling
	SymFmt string
	// parse param from opcode
	Parse func(opcode uint16) InstructionParm
	// print decompiled instruction
	Print func(inst Instruction, parm InstructionParm) string
	// exec
	Exec func(chip *Chip8, parm InstructionParm)
}

//All opcodes
var istset = []Instruction{
	{0x00EE, 0xFFFF, "RET", nil, print, func(chip *Chip8, parm InstructionParm) {
		chip.pc = chip.stack.pop()
		chip.pc += 2
	}},
	{0x00E0, 0xFFFF, "CLS", nil, print, func(chip *Chip8, parm InstructionParm) {
		for idx, _ := range chip.VideoMemory {
			chip.VideoMemory[idx] = 0
		}
		chip.pc += 2
	}},
	{0x0000, 0xF000, "SYS 0x%03X", parseAddr, printAddr, nil},
	{0x1000, 0xF000, "JP 0x%03X", parseAddr, printAddr, func(chip *Chip8, parm InstructionParm) {
		chip.pc = parm.Addr
	}},
	{0x2000, 0xF000, "CALL 0x%03X", parseAddr, printAddr, func(chip *Chip8, parm InstructionParm) {
		chip.stack.push(chip.pc)
		chip.pc = parm.Addr
	}},
	{0x3000, 0xF000, "SE V%X, 0x%02X", parseRegAndByte, printRegAndByte, func(chip *Chip8, parm InstructionParm) {
		if chip.Register[parm.Vx] == parm.Byte {
			chip.pc += 2
		}
		chip.pc += 2
	}},
	{0x4000, 0xF000, "SNE V%X, 0x%02X", parseRegAndByte, printRegAndByte, func(chip *Chip8, parm InstructionParm) {
		if chip.Register[parm.Vx] != parm.Byte {
			chip.pc += 2
		}
		chip.pc += 2
	}},
	{0x5000, 0xF000, "SE V%X {, V%X}", parse2Reg, print2Reg, func(chip *Chip8, parm InstructionParm) {
		if chip.Register[parm.Vx] == chip.Register[parm.Vy] {
			chip.pc += 2
		}
		chip.pc += 2
	}},
	{0x6000, 0xF000, "LD V%X, 0x%02X", parseRegAndByte, printRegAndByte, func(chip *Chip8, parm InstructionParm) {
		chip.Register[parm.Vx] = parm.Byte
		chip.pc += 2
	}},
	{0x7000, 0xF000, "ADD V%X, 0x%02X", parseRegAndByte, printRegAndByte, func(chip *Chip8, parm InstructionParm) {
		chip.Register[parm.Vx] += parm.Byte
		chip.pc += 2
	}},
	{0x8000, 0xF00F, "LD V%X, V%X", parse2Reg, print2Reg, func(chip *Chip8, parm InstructionParm) {
		chip.Register[parm.Vx] = chip.Register[parm.Vy]
		chip.pc += 2
	}},
	{0x8001, 0xF00F, "OR V%X, V%X", parse2Reg, print2Reg, func(chip *Chip8, parm InstructionParm) {
		chip.Register[parm.Vx] |= chip.Register[parm.Vy]
		chip.pc += 2
	}},
	{0x8002, 0xF00F, "AND V%X, V%X", parse2Reg, print2Reg, func(chip *Chip8, parm InstructionParm) {
		chip.Register[parm.Vx] &= chip.Register[parm.Vy]
		chip.pc += 2
	}},
	{0x8003, 0xF00F, "XOR V%X, V%X", parse2Reg, print2Reg, func(chip *Chip8, parm InstructionParm) {
		chip.Register[parm.Vx] ^= chip.Register[parm.Vy]
		chip.pc += 2
	}},
	{0x8004, 0xF00F, "ADD V%X, V%X", parse2Reg, print2Reg, func(chip *Chip8, parm InstructionParm) {
		chip.Register[15] = 0
		if uint16(chip.Register[parm.Vx]+chip.Register[parm.Vy]) > 255 {
			chip.Register[15] = 1
		}
		chip.Register[parm.Vx] += chip.Register[parm.Vy]
		chip.pc += 2
	}},
	{0x8005, 0xF00F, "SUB V%X, V%X", parse2Reg, print2Reg, func(chip *Chip8, parm InstructionParm) {
		chip.Register[15] = 0
		if chip.Register[parm.Vx] > chip.Register[parm.Vy] {
			chip.Register[15] = 1
		}
		chip.Register[parm.Vx] = chip.Register[parm.Vx] - chip.Register[parm.Vy]
		chip.pc += 2
	}},
	{0x8006, 0xF00F, "SHR V%X {, V%X}", parse2Reg, print2Reg, func(chip *Chip8, parm InstructionParm) {
		chip.Register[15] = chip.Register[parm.Vx] & 0x01
		chip.Register[parm.Vx] = chip.Register[parm.Vx] >> 1
		chip.pc += 2
	}},
	{0x8007, 0xF00F, "SUBN V%X, V%X", parse2Reg, print2Reg, func(chip *Chip8, parm InstructionParm) {
		chip.Register[15] = 0
		if chip.Register[parm.Vy] > chip.Register[parm.Vx] {
			chip.Register[15] = 1
		}
		chip.Register[parm.Vx] = chip.Register[parm.Vy] - chip.Register[parm.Vx]
		chip.pc += 2
	}},
	{0x800E, 0xF00F, "SHL V%X {, V%X}", parse2Reg, print2Reg, func(chip *Chip8, parm InstructionParm) {
		chip.Register[15] = chip.Register[parm.Vx] >> 7
		chip.Register[parm.Vx] = chip.Register[parm.Vx] << 1
		chip.pc += 2
	}},
	{0x9000, 0xF000, "SNE V%X, V%X", parse2Reg, print2Reg, func(chip *Chip8, parm InstructionParm) {
		if chip.Register[parm.Vx] != chip.Register[parm.Vy] {
			chip.pc += 2
		}
		chip.pc += 2
	}},
	{0xA000, 0xF000, "LD I, 0x%03X", parseAddr, printAddr, func(chip *Chip8, parm InstructionParm) {
		chip.I = parm.Addr
		chip.pc += 2
	}},
	{0xB000, 0xF000, "JP V0, 0x%03X", parseAddr, printAddr, func(chip *Chip8, parm InstructionParm) {
		chip.pc = uint16(chip.Register[0]) + parm.Addr
	}},
	{0xC000, 0xF000, "RND V%X, 0x%02X", parseRegAndByte, printRegAndByte, func(chip *Chip8, parm InstructionParm) {
		chip.Register[parm.Vx] = uint8(rand.Intn(255)) & parm.Byte
		chip.pc += 2
	}},
	{0xD000, 0xF000, "DRW V%X, V%X, %X", parse2RegAndNibble, print2RegAndNibble, func(chip *Chip8, parm InstructionParm) {
		// loop

		for idx := uint8(0); idx < parm.Byte; idx++ {
			//video memory location
			y := chip.Register[parm.Vy] + idx
			/*if ( y > 31							 ) {
				y = 0
			}*/
			vidlochi := (y * 8) + chip.Register[parm.Vx]/8
			vidloclo := vidlochi + 1
			if vidloclo == (y+1)*8 {
				vidloclo = y * 8
			}

			// load line of sprite
			spr := chip.memory[chip.I+uint16(idx)]

			//shift sprite according to x coordinates
			sprhi := (spr >> (chip.Register[parm.Vx] % 8))
			sprlo := (spr << (8 - chip.Register[parm.Vx]%8))

			//check collitions
			chip.Register[15] = 0
			if (chip.VideoMemory[vidlochi]^sprhi != chip.VideoMemory[vidlochi]|sprhi) ||
				(chip.VideoMemory[vidloclo]^sprlo != chip.VideoMemory[vidloclo]|sprlo) {
				chip.Register[15] = 1
			}
			// xor video memory with sprite
			chip.VideoMemory[vidlochi] = chip.VideoMemory[vidlochi] ^ sprhi
			chip.VideoMemory[vidloclo] = chip.VideoMemory[vidloclo] ^ sprlo

		}
		chip.pc += 2
	}},
	{0xE09E, 0xF0FF, "SKP V%X", parseReg, printReg, func(chip *Chip8, parm InstructionParm) {
		// check keyboard down
		if chip.keyboard[chip.Register[parm.Vx]] == 1 {
			chip.pc += 2
		}
		// reset keyboard
		chip.keyboard[chip.Register[parm.Vx]] = 0
		chip.pc += 2
	}},
	{0xE0A1, 0xF0FF, "SKNP V%X", parseReg, printReg, func(chip *Chip8, parm InstructionParm) {
		// check keyboard up
		if chip.keyboard[chip.Register[parm.Vx]] == 0 {
			chip.pc += 2
		}
		// reset keyboard
		chip.keyboard[chip.Register[parm.Vx]] = 0
		chip.pc += 2
	}},
	{0xF00A, 0xF00F, "LD V%X, K", parseReg, printReg, func(chip *Chip8, parm InstructionParm) {
		// check keyboard
		for idx, value := range chip.keyboard {
			if value != 0 {
				chip.Register[parm.Vx] = uint8(idx)
				chip.pc += 2
			}
		}
	}},
	{0xF007, 0xF00F, "LD V%X, DT", parseReg, printReg, func(chip *Chip8, parm InstructionParm) {
		chip.Register[parm.Vx] = chip.Delay
		chip.pc += 2
	}},

	{0xF01E, 0xF0FF, "ADD I, V%X", parseReg, printReg, func(chip *Chip8, parm InstructionParm) {
		chip.I += uint16(chip.Register[parm.Vx])
		chip.pc += 2
	}},
	{0xF018, 0xF0FF, "LD ST, V%X", parseReg, printReg, func(chip *Chip8, parm InstructionParm) {
		chip.Sound = chip.Register[parm.Vx]
		chip.pc += 2
	}},
	{0xF015, 0xF0FF, "LD DT, V%X", parseReg, printReg, func(chip *Chip8, parm InstructionParm) {
		chip.Delay = chip.Register[parm.Vx]
		chip.pc += 2
	}},
	{0xF065, 0xF0FF, "LD V%X, [I]", parseReg, printReg, func(chip *Chip8, parm InstructionParm) {
		for idx := uint16(0); idx <= uint16(parm.Vx); idx++ {
			chip.Register[idx] = chip.memory[chip.I+idx]
		}
		chip.pc += 2
	}},
	{0xF055, 0xF0FF, "LD [I], V%X", parseReg, printReg, func(chip *Chip8, parm InstructionParm) {
		for idx := uint16(0); idx <= uint16(parm.Vx); idx++ {
			chip.memory[chip.I+idx] = chip.Register[idx]
		}
		chip.pc += 2
	}},

	{0xF033, 0xF0FF, "LD B, V%X", parseReg, printReg, func(chip *Chip8, parm InstructionParm) {
		// store BCD of Vx into I, I+1, I+2
		value := chip.Register[parm.Vx]
		chip.memory[chip.I] = value / 100
		value = value - chip.memory[chip.I]*100
		chip.memory[chip.I+1] = value / 10
		value = value - chip.memory[chip.I+1]*10
		chip.memory[chip.I+2] = value
		chip.pc += 2
	}},
	{0xF029, 0xF0FF, "LD F, V%X", parseReg, printReg, func(chip *Chip8, parm InstructionParm) {
		// set address of sprite for char in VX
		// these sprites are loaded at memory location 0x000
		chip.I = 0x000 + uint16((chip.Register[parm.Vx] * 5))
		chip.pc += 2
	}},
}

// various print functions
func printAddr(inst Instruction, parm InstructionParm) string {
	return fmt.Sprintf(inst.SymFmt, parm.Addr)
}

func print(inst Instruction, parm InstructionParm) string {
	return fmt.Sprintf(inst.SymFmt)
}

func printReg(inst Instruction, parm InstructionParm) string {
	return fmt.Sprintf(inst.SymFmt, parm.Vx)
}

func print2Reg(inst Instruction, parm InstructionParm) string {
	return fmt.Sprintf(inst.SymFmt, parm.Vx, parm.Vy)
}

func printRegAndByte(inst Instruction, parm InstructionParm) string {
	return fmt.Sprintf(inst.SymFmt, parm.Vx, parm.Byte)
}

func print2RegAndNibble(inst Instruction, parm InstructionParm) string {
	return fmt.Sprintf(inst.SymFmt, parm.Vx, parm.Vy, parm.Byte)
}

// Parse one params in form of ?nnn where
//	nnn => 12bit value
func parseAddr(opcode uint16) InstructionParm {
	return InstructionParm{
		Addr: opcode & 0x0FFF,
	}
}

//	Parse 3 params in form of ?xyn where
//		x => 4 bit registry number
//		y => 4 bit registry number
//		n => 4 bit nibble
func parse2RegAndNibble(opcode uint16) InstructionParm {
	return InstructionParm{
		Vx:   uint8((opcode & 0x0F00) / 256),
		Vy:   uint8((opcode & 0x00F0) / 16),
		Byte: uint8((opcode & 0x000F)),
	}
}

//	Parse 2 params in form of ?xkk where
//		x => 4 bit registry number
//		kk => 8 bit value
func parseRegAndByte(opcode uint16) InstructionParm {
	return InstructionParm{
		Vx:   uint8((opcode & 0x0F00) / 256),
		Byte: uint8((opcode & 0x00FF)),
	}
}

func parseReg(opcode uint16) InstructionParm {
	return InstructionParm{
		Vx: uint8((opcode & 0x0F00) / 256),
	}
}

func parse2Reg(opcode uint16) InstructionParm {
	return InstructionParm{
		Vx: uint8((opcode & 0x0F00) / 256),
		Vy: uint8((opcode & 0x00F0) / 16),
	}
}

func (chip *Chip8) findOp(opCode uint16) (Instruction, error) {
	// scan for instrunction by bitmasking opcode
	for _, value := range istset {
		if opCode&value.Bitmask == value.Opcode { // founded a valid instruction
			return value, nil
		}
	}
	return Instruction{}, errors.New("Invalid opcode")
}

func (chip *Chip8) videoMemoryDump() {

	fmt.Printf("%02X:", 0)
	for idx, cell := range chip.VideoMemory {
		fmt.Printf("%08b", cell)
		if (idx+1)%8 == 0 {
			fmt.Printf("\n%02X:", idx)
		}
	}
	fmt.Print("\n")

}

// Decompile loaded program
func (chip *Chip8) Decompile() {
	for chip.pc < chip.prgEnd {
		// load the 2 byte opcode
		curOp := uint16(chip.memory[chip.pc])*256 + uint16(chip.memory[chip.pc+1])
		// dump memory
		fmt.Printf("0x%04X: %04X ", chip.pc, curOp)
		// decode func
		cmd, err := chip.findOp(curOp)
		if err == nil { // founded command
			if cmd.Parse != nil { // print decompiled opcode
				fmt.Print(cmd.Print(cmd, cmd.Parse(curOp)))
			} else {
				fmt.Print(cmd.Print(cmd, InstructionParm{}))
			}
		}

		fmt.Print("\n")
		// increment program counter
		chip.pc += 2
	}
}

// Execute a single instruction and increment program counter
// return a boolean that is true if there are more instruction to execute, and the string that represent the opcode executed (useful for debugging purpose)
func (chip *Chip8) Run() (bool, string) {
	if chip.pc < chip.prgEnd {
		// load the 2 byte opcode
		curOp := uint16(chip.memory[chip.pc])*256 + uint16(chip.memory[chip.pc+1])

		// decode func
		cmd, err := chip.findOp(curOp)
		if err == nil { // founded command
			parm := InstructionParm{}
			if cmd.Parse != nil {
				parm = cmd.Parse(curOp)
			}
			cmddump := fmt.Sprintf("0x%04X: %04X %s", chip.pc, curOp, cmd.Print(cmd, parm))
			//exec instruction
			cmd.Exec(chip, parm)
			return true, cmddump
		} else {
			panic("Invalid opcode")
		}
	}
	//end of program
	return false, ""
}

// Inizialize the interpreter
func (chip *Chip8) Init() {
	fmt.Println("Loading fonts")
	font := []byte{0xF0, 0x90, 0x90, 0x90, 0xF0, // sprite for char '0'
		0x20, 0x60, 0x20, 0x20, 0x70, // sprite for char '1'
		0xF0, 0x10, 0xF0, 0x80, 0xF0, // sprite for char '2'
		0xF0, 0x10, 0xF0, 0x10, 0xF0, // sprite for char '3'
		0x90, 0x90, 0xF0, 0x10, 0x10, // sprite for char '4'
		0xF0, 0x80, 0xF0, 0x10, 0xF0, // sprite for char '5'
		0xF0, 0x80, 0xF0, 0x90, 0xF0, // sprite for char '6'
		0xF0, 0x10, 0x20, 0x40, 0x40, // sprite for char '7'
		0xF0, 0x90, 0xF0, 0x90, 0xF0, // sprite for char '8'
		0xF0, 0x90, 0xF0, 0x10, 0xF0, // sprite for char '9'
		0xF0, 0x90, 0xF0, 0x90, 0x90, // sprite for char 'A'
		0xE0, 0x90, 0xE0, 0x90, 0xE0, // sprite for char 'B'
		0xF0, 0x80, 0x80, 0x80, 0xF0, // sprite for char 'C'
		0xE0, 0x90, 0x90, 0x90, 0xE0, // sprite for char 'D'
		0xF0, 0x80, 0xF0, 0x80, 0xF0, // sprite for char 'E'
		0xF0, 0x80, 0xF0, 0x80, 0x80, // sprite for char 'F'
	}
	// load fonts
	copy(chip.memory[:], font)

	//set a sane random seed
	rand.Seed(time.Now().UnixNano())

	//create stack
	chip.stack = &stack{}

	//start 60hertz clock
	go func() {
		for {
			if chip.Delay > 0 {
				chip.Delay--
			}
			if chip.Sound > 0 {
				chip.Sound--
			}
			time.Sleep(16 * time.Millisecond)
		}
	}()

}

// Load program returning the number of bytes loaded
func (chip *Chip8) Load(buffer []byte) int {
	x := copy(chip.memory[512:], buffer)
	chip.pc = 512 // starting address for programs
	chip.prgEnd = 512 + uint16(x)
	fmt.Printf("Loaded %d bytes\n", x)
	return x
}

// tell the interpreter that some key are pressed
func (chip *Chip8) KeyPressed(key rune) {
	if key == '9' {
		for idx, _ := range chip.keyboard {
			chip.keyboard[idx] = 0
		}
	}
	switch key {
	case '0':
		chip.keyboard[0x00] = 1
	case '1':
		chip.keyboard[0x01] = 1
	case '2':
		chip.keyboard[0x02] = 1
	case '3':
		chip.keyboard[0x03] = 1
	case '4':
		chip.keyboard[0x04] = 1
	case '5':
		chip.keyboard[0x05] = 1
	case '6':
		chip.keyboard[0x06] = 1
	case '7':
		chip.keyboard[0x07] = 1
	case '8':
		chip.keyboard[0x08] = 1
	case '9':
		chip.keyboard[0x09] = 1
	case 'A':
		chip.keyboard[0x0A] = 1
	case 'B':
		chip.keyboard[0x0B] = 1
	case 'C':
		chip.keyboard[0x0C] = 1
	case 'D':
		chip.keyboard[0x0D] = 1
	case 'E':
		chip.keyboard[0x0E] = 1
	case 'F':
		chip.keyboard[0x0F] = 1
	}
}
