##About
Ghip8 is a golang learning project that implement a chip8 interpreter/disassembler useful for developing a complete emulator.
WARNING: the interpreter is incomplete, some opcodes aren't implemented yet so keep in mind that only few roms will work for now.

##Example
```
	chip8 := &Chip8{}
	
	fmt.Printf("Loading file %s\n", filename)	
	if buffer, err := ioutil.ReadFile(*filename); err == nil {
		chip8.Load(buffer)
		for chip8.Run() {}		
	} else {
		fmt.Printf("Error loading file %s",err)
	}
```

