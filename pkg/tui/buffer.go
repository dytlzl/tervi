package tui

import "github.com/dytlzl/tervi/pkg/key"

func readBuffer(buffer []rune) (rune, int) {
	if len(buffer) == 0 {
		return 0, 0
	}
	if len(buffer) > 2 {
		v := buffer[0]*0x10000 + buffer[1]*0x100 + buffer[2]
		switch v {
		// ^[[A
		case key.ArrowUp, key.ArrowDown, key.ArrowRight, key.ArrowLeft:
			return v, 3
		// ^[OA
		case key.ArrowUp - 0xc00, key.ArrowDown - 0xc00, key.ArrowRight - 0xc00, key.ArrowLeft - 0xc00:
			return v + 0xc00, 3
		}
	}
	return buffer[0], 1
}
