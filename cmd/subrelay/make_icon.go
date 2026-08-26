//go:build ignore

// make_icon.go converts the tray icon PNG to a Windows ICO file and
// compiles a resource file (.rc) into a .syso that embeds the ICO as
// the executable icon. Run with:
//
//	go run make_icon.go
package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	pngPath := filepath.Join("..", "..", "internal", "tray", "icon.png")
	icoPath := "icon.ico"
	rcPath := "resource.rc"
	sysoPath := "resource.syso"

	pngData, err := os.ReadFile(pngPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read png: %v\n", err)
		os.Exit(1)
	}

	// Build ICO file with a single PNG-encoded image.
	// ICONDIR: reserved(2)=0, type(2)=1, count(2)=1
	header := make([]byte, 6)
	binary.LittleEndian.PutUint16(header[0:2], 0) // reserved
	binary.LittleEndian.PutUint16(header[2:4], 1) // type = ICO
	binary.LittleEndian.PutUint16(header[4:6], 1) // count = 1

	// ICONDIRENTRY (16 bytes):
	// width(1)=32, height(1)=32, colors(1)=0, reserved(1)=0,
	// planes(2)=1, bitcount(2)=32, bytesinres(4)=len(png),
	// imageoffset(4)=22 (6+16)
	entry := make([]byte, 16)
	entry[0] = 32 // width (32px; 0 means 256)
	entry[1] = 32 // height
	entry[2] = 0  // color count (0 = no palette)
	entry[3] = 0  // reserved
	binary.LittleEndian.PutUint16(entry[4:6], 1)             // planes
	binary.LittleEndian.PutUint16(entry[6:8], 32)            // bit count
	binary.LittleEndian.PutUint32(entry[8:12], uint32(len(pngData))) // image size
	binary.LittleEndian.PutUint32(entry[12:16], 22)          // offset to image data

	ico := make([]byte, 0, len(header)+len(entry)+len(pngData))
	ico = append(ico, header...)
	ico = append(ico, entry...)
	ico = append(ico, pngData...)

	if err := os.WriteFile(icoPath, ico, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write ico: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("wrote", icoPath)

	// Write resource file.
	rcContent := `// Resource file for Windows executable icon.
1 ICON "icon.ico"
`
	if err := os.WriteFile(rcPath, []byte(rcContent), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write rc: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("wrote", rcPath)

	// Compile with windres. Add MinGW to PATH if not already there.
	path := os.Getenv("Path")
	mingwBin := `C:\ProgramData\mingw64\mingw64\bin`
	windresPath := filepath.Join(mingwBin, "windres.exe")
	if _, err := os.Stat(windresPath); err != nil {
		// Try to find windres on PATH
		cmd := exec.Command("windres", "--version")
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "windres not found: %v\n", err)
			os.Exit(1)
		}
		windresPath = "windres"
	} else {
		os.Setenv("Path", mingwBin+string(os.PathListSeparator)+path)
	}

	cmd := exec.Command(windresPath, "-o", sysoPath, rcPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "windres: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("wrote", sysoPath)
}
