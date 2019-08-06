/*

	Infro from
	http://devernay.free.fr/hacks/chip8/C8TECH10.HTM#1.0

*/

package ghip8

import (
	"errors"
	"fmt"
	"math/rand"
)

type Chip8 struct {
	// accessible registers
	Memory [4096]uint8

	Register     [16]uint8 // V0 - VF
	I            uint16    // I register
	Delay, Sound uint8     //Sound and delay counters

	//internal registers
	Pc          uint16 // Program counter
	Sp          uint8  //stack pointer
	Stack       [16]uint16
	VideoMemory [256]uint8 // 64x32 pixel video

	//program lenght
	PrgEnd uint16
}

// Instruction params
type InstructionParm struct {
	Addr uint16 //address 16bit
	Byte uint8  // byte
	Vx   uint8  // reg 1
	Vy   uint8  // reg 2
}

// Instruction struct
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
	Print func(inst Instruction, parm InstructionParm)
	//exec
	Exec func(chip *Chip8, parm InstructionParm)
}

/*
	All opcodes
*/
var istset = []Instruction{
	{0x00EE, 0xFFFF, "RET", nil, print, nil},
	{0x00E0, 0xFFFF, "CLS", nil, print, nil},
	{0x0000, 0xF000, "SYS 0x%03X", parseAddr, printAddr, nil},
	{0x1000, 0xF000, "JP 0x%03X", parseAddr, printAddr, func(chip *Chip8, parm InstructionParm) {
									chip.Pc = parm.Addr
								}},
	{0x2000, 0xF000, "CALL 0x%03X", parseAddr, printAddr, nil},
	{0x3000, 0xF000, "SE V%X, 0x%02X", parseRegAndByte, printRegAndByte,func(chip *Chip8, parm InstructionParm) {
									if chip.Register[parm.Vx] == parm.Byte {
										chip.Pc += 2
									}
									chip.Pc += 2
								}},
	{0x4000, 0xF000, "SNE V%X, 0x%02X", parseRegAndByte, printRegAndByte, func(chip *Chip8, parm InstructionParm) {
									if chip.Register[parm.Vx] != parm.Byte {
										chip.Pc += 2
									}
									chip.Pc += 2
								}},
	{0x5000, 0xF000, "SE V%X {, V%X}", parse2Reg, print2Reg, func(chip *Chip8, parm InstructionParm) {
									if chip.Register[parm.Vx] == chip.Register[parm.Vy] {
										chip.Pc += 2
									}
									chip.Pc += 2
								}},
	{0x6000, 0xF000, "LD V%X, 0x%02X", parseRegAndByte, printRegAndByte, func(chip *Chip8, parm InstructionParm) {
									chip.Register[parm.Vx] = parm.Byte
									chip.Pc += 2
								}},
	{0x7000, 0xF000, "ADD V%X, 0x%02X", parseRegAndByte, printRegAndByte, func(chip *Chip8, parm InstructionParm) {
									chip.Register[parm.Vx] += parm.Byte
									chip.Pc += 2
								}},
	{0x8000, 0xF00F, "LD V%X, V%X", parse2Reg, print2Reg, func(chip *Chip8, parm InstructionParm) {
									chip.Register[parm.Vx] = chip.Register[parm.Vy]
									chip.Pc += 2
								}},
	{0x8001, 0xF00F, "OR V%X, V%X", parse2Reg, print2Reg, func(chip *Chip8, parm InstructionParm) {
									chip.Register[parm.Vx] |= chip.Register[parm.Vy]
									chip.Pc += 2
								}},
	{0x8002, 0xF00F, "AND V%X, V%X", parse2Reg, print2Reg, func(chip *Chip8, parm InstructionParm) {
									chip.Register[parm.Vx] &= chip.Register[parm.Vy]
									chip.Pc += 2
								}},
	{0x8003, 0xF00F, "XOR V%X, V%X", parse2Reg, print2Reg, func(chip *Chip8, parm InstructionParm) {
									chip.Register[parm.Vx] ^= chip.Register[parm.Vy]
									chip.Pc += 2
								}},
	{0x8004, 0xF00F, "ADD V%X, V%X", parse2Reg, print2Reg, func(chip *Chip8, parm InstructionParm) {
									chip.Register[15]=0
									chip.Register[parm.Vx] += chip.Register[parm.Vy]
									if (chip.Register[parm.Vx] > 255) {
										chip.Register[15]=1
										chip.Register[parm.Vx] &= 0xFF
									}
									chip.Pc += 2
								}},
	{0x8005, 0xF00F, "SUB V%X, V%X", parse2Reg, print2Reg, nil},
	{0x8006, 0xF00F, "SHR V%X {, V%X}", parse2Reg, print2Reg, func(chip *Chip8, parm InstructionParm) {
									chip.Register[15] = chip.Register[parm.Vx] & 0x01
									chip.Register[parm.Vx] = chip.Register[parm.Vx] >> 1
									chip.Pc += 2
								}},
	{0x8007, 0xF00F, "SUBN V%X, V%X", parse2Reg, print2Reg, nil},
	{0x800E, 0xF00F, "SHL V%X {, V%X}", parse2Reg, print2Reg, func(chip *Chip8, parm InstructionParm) {
									chip.Register[15] = chip.Register[parm.Vx] >> 7
									chip.Register[parm.Vx] = chip.Register[parm.Vx] << 1
									chip.Pc += 2
								}},
	{0x9000, 0xF000, "SNE V%X, V%X", parse2Reg, print2Reg, func(chip *Chip8, parm InstructionParm) {
									if chip.Register[parm.Vx] != chip.Register[parm.Vy] {
										chip.Pc += 2
									}
									chip.Pc += 2
								}},
	{0xA000, 0xF000, "LD I, 0x%03X", parseAddr, printAddr, func(chip *Chip8, parm InstructionParm) {
									chip.I = parm.Addr
									chip.Pc += 2
								}},
	{0xB000, 0xF000, "JP V0, 0x%03X", parseAddr, printAddr, func(chip *Chip8, parm InstructionParm) {
									chip.Pc = uint16(chip.Register[0]) + parm.Addr
								}},
	{0xC000, 0xF000, "RND V%X, 0x%02X", parseRegAndByte, printRegAndByte, func(chip *Chip8, parm InstructionParm) {
									chip.Register[parm.Vx] = uint8(rand.Intn(255)) & parm.Byte
									chip.Pc += 2
								}},
	{0xD000, 0xF000, "DRW V%X, V%X, %X", parse2RegAndNibble, print2RegAndNibble, func(chip *Chip8, parm InstructionParm) {
									// loop
									for idx := uint8(0); idx < parm.Byte; idx++ {
										//video memory location
										vidloc := ((chip.Register[parm.Vy] + uint8(idx)) * 8) + 
												chip.Register[parm.Vx]/8

										// load line of sprite
										spr := chip.Memory[chip.I+uint16(idx)]

										//shift sprite according to x coordinates
										sprhi := (spr >> (chip.Register[parm.Vx] % 8))
										sprlo := (spr << (8 - chip.Register[parm.Vx]%8))

										// xor video memory with sprite
										chip.VideoMemory[vidloc] = chip.VideoMemory[vidloc] ^ sprhi
										chip.VideoMemory[vidloc+1] = chip.VideoMemory[vidloc+1] ^ sprlo
										//dump video memory
										chip.videoMemoryDump()

									}
									chip.Pc += 2
								}},
	{0xE09E, 0xF0FF, "SKP V%X", parseReg, printReg, nil},
	{0xE0A1, 0xF0FF, "SKNP V%X", parseReg, printReg, nil},
	{0xF00A, 0xF00F, "LD V%X, K", parseReg, printReg, nil},
	{0xF007, 0xF00F, "LD V%X, DT", parseReg, printReg, func(chip *Chip8, parm InstructionParm) {
									chip.Register[parm.Vx] = chip.Delay
									chip.Pc += 2
								}},
	{0xF01E, 0xF0FF, "ADD I, V%X", parseReg, printReg, func(chip *Chip8, parm InstructionParm) {
									chip.I += uint16(chip.Register[parm.Vx])
									chip.Pc += 2
								}},
	{0xF018, 0xF0FF, "LD ST, V%X", parseReg, printReg, func(chip *Chip8, parm InstructionParm) {
									chip.Sound = chip.Register[parm.Vx]
									chip.Pc += 2
								}},
	{0xF015, 0xF0FF, "LD DT, V%X", parseReg, printReg, func(chip *Chip8, parm InstructionParm) {
									chip.Delay = chip.Register[parm.Vx]
									chip.Pc += 2
								}},
	{0xF065, 0xF0FF, "LD V%X, [I]", parseReg, printReg, func(chip *Chip8, parm InstructionParm) {
									for idx := uint16(0); idx < uint16(parm.Vx); idx++ {
										chip.Register[idx] = chip.Memory[chip.I+idx]
									}
									chip.Pc += 2
								}},
	{0xF055, 0xF0FF, "LD [I], V%X", parseReg, printReg, func(chip *Chip8, parm InstructionParm) {
									for idx := uint16(0); idx < uint16(parm.Vx); idx++ {
										chip.Memory[chip.I+idx] = chip.Register[idx] 
									}
									chip.Pc += 2
								}},
	{0xF033, 0xF0FF, "LD B, V%X", parseReg, printReg, nil},
	{0xF029, 0xF0FF, "LD F, V%X", parseReg, printReg, nil},
}

/*
	various print functions
*/
func printAddr(inst Instruction, parm InstructionParm) {
	fmt.Printf(inst.SymFmt, parm.Addr)
}

func print(inst Instruction, parm InstructionParm) {
	fmt.Printf(inst.SymFmt)
}

func printReg(inst Instruction, parm InstructionParm) {
	fmt.Printf(inst.SymFmt, parm.Vx)
}

func print2Reg(inst Instruction, parm InstructionParm) {
	fmt.Printf(inst.SymFmt, parm.Vx, parm.Vy)
}

func printRegAndByte(inst Instruction, parm InstructionParm) {
	fmt.Printf(inst.SymFmt, parm.Vx, parm.Byte)
}

func print2RegAndNibble(inst Instruction, parm InstructionParm) {
	fmt.Printf(inst.SymFmt, parm.Vx, parm.Vy, parm.Byte)
}

/*
	Parse one params in form of ?nnn where
		nnn => 12bit value
*/
func parseAddr(opcode uint16) InstructionParm {
	return InstructionParm{
		Addr: opcode & 0x0FFF,
	}
}

/*
	Parse 3 params in form of ?xyn where
		x => 4 bit registry number
		y => 4 bit registry number
		n => 4 bit nibble

*/
func parse2RegAndNibble(opcode uint16) InstructionParm {
	return InstructionParm{
		Vx:   uint8((opcode & 0x0F00) / 256),
		Vy:   uint8((opcode & 0x00F0) / 16),
		Byte: uint8((opcode & 0x000F)),
	}
}

/*
	Parse 2 params in form of ?xkk where
		x => 4 bit registry number
		kk => 8 bit value

*/
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
		if opCode & value.Bitmask == value.Opcode { // founded a valid instruction
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

/*
	Decompile chip8 memory
*/
func (chip *Chip8) Decompile() {
	for chip.Pc < chip.PrgEnd {
		// load the 2 byte opcode
		curOp := uint16(chip.Memory[chip.Pc])*256 + uint16(chip.Memory[chip.Pc+1])
		// dump memory
		fmt.Printf("0x%04X: %04X ", chip.Pc, curOp)
		// decode func
		cmd, err := chip.findOp(curOp)
		if err == nil { // founded command
			if cmd.Parse != nil { // print decompiled opcode
				cmd.Print(cmd, cmd.Parse(curOp))
			} else {
				cmd.Print(cmd, InstructionParm{})
			}
		}

		fmt.Print("\n")
		// increment program counter
		chip.Pc += 2
	}
}

/*
	Run chip8 programs

	return true if there are more instruction to execute, otherwise return false
*/
func (chip *Chip8) Run() bool {
	if chip.Pc < chip.PrgEnd {
		// load the 2 byte opcode
		curOp := uint16(chip.Memory[chip.Pc])*256 + uint16(chip.Memory[chip.Pc+1])
		// dump memory
		fmt.Printf("0x%04X: %04X ", chip.Pc, curOp)
		// decode func
		cmd, err := chip.findOp(curOp)
		if err == nil { // founded command
			parm := InstructionParm{}
			if cmd.Parse != nil {
				parm = cmd.Parse(curOp)
			} 
			cmd.Print(cmd, parm)
			//exec instruction
			cmd.Exec(chip, parm)
		} else {
			panic("Invalid opcode")
		}

		fmt.Print("\n")
		
		return true
	}
	//end of program
	return false
}

func (chip *Chip8) Load(buffer []byte) {
	x := copy(chip.Memory[512:], buffer)
	chip.Pc = 512 // starting address for programs
	chip.PrgEnd = 512 + uint16(x)
	fmt.Printf("Loaded %d bytes\n", x)
}
