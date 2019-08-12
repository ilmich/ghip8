# About
Ghip8 is a golang learning project that implement a chip8 interpreter/disassembler useful for developing a complete emulator.

# Example
```
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	
	"github.com/ilmich/ghip8"
)

func main() {

	var filename = flag.String("f", "", "Chip8 source file")
	var decompile = flag.Bool("d", false, "Decompile file")	
	
	flag.Parse()

	chip8 := &ghip8.Chip8{}
	
	fmt.Printf("Loading file %s\n", filename)
	
	
	if buffer, err := ioutil.ReadFile(*filename); err == nil {
		chip8.Init()
		chip8.Load(buffer)
		if *decompile {
			chip8.Decompile()
		}else {			
			for chip8.Run() {
				// cool stuffs of your emulator
				// refresh the ui by reading chip8 video memory
				// sounds by reading sound counter
				// and so on
			}
		}
	} else {
		fmt.Printf("Error loading file %s",err)
	}
	

}

```

