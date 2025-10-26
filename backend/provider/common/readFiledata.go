package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"main/structs"
	"os"
	"strings"
)

func ReadFileData(file *os.File, inode structs.Inode, blockPos int32) (string, error) {
	var content strings.Builder
	totalBytesRead := 0

	fmt.Printf("DEBUG readUsersFileData - blockPos: %d, inode.I_size: %d\n", blockPos, inode.I_size)

	// 1. Leer bloques directos (slots 0-11)
	for i := 0; i < 12 && inode.I_block[i] != 0 && totalBytesRead < int(inode.I_size); i++ {
		blockNum := inode.I_block[i]
		fmt.Printf("DEBUG: Leyendo bloque directo %d (número: %d)\n", i, blockNum)

		bytesRead, err := readDirectBlock(file, blockNum, blockPos, int(inode.I_size)-totalBytesRead, &content)
		if err != nil {
			return "", err
		}
		totalBytesRead += bytesRead
	} // 2. Leer bloque indirecto simple (slot 12)
	if inode.I_block[12] != 0 && totalBytesRead < int(inode.I_size) {
		bytesRead, err := readIndirectBlock(file, inode.I_block[12], blockPos, int(inode.I_size)-totalBytesRead, &content)
		if err != nil {
			return "", err
		}
		totalBytesRead += bytesRead
	}

	// 3. Leer bloque doblemente indirecto (slot 13)
	if inode.I_block[13] != 0 && totalBytesRead < int(inode.I_size) {
		bytesRead, err := readDoubleIndirectBlock(file, inode.I_block[13], blockPos, int(inode.I_size)-totalBytesRead, &content)
		if err != nil {
			return "", err
		}
		totalBytesRead += bytesRead
	}

	result := content.String()

	// Truncar al tamaño exacto del archivo
	if len(result) > int(inode.I_size) {
		result = result[:inode.I_size]
	}

	return result, nil
}

// readDirectBlock lee un bloque directo de datos
func readDirectBlock(file *os.File, blockNum int32, blockPos int32, maxBytes int, content *strings.Builder) (int, error) {
	// CRÍTICO: Si blockPos es 0, significa que S_block_start está en formato relativo
	// Necesitamos obtener el superbloque para calcular la posición absoluta
	blockPosition := int64(blockPos + (blockNum * structs.BLOCK_SIZE))

	fmt.Printf("DEBUG readDirectBlock: blockNum=%d, blockPos=%d, posición calculada=%d\n",
		blockNum, blockPos, blockPosition)

	// Leer el bloque completo
	blockBytes := make([]byte, structs.BLOCK_SIZE)
	bytesRead, err := file.ReadAt(blockBytes, blockPosition)
	if err != nil && bytesRead == 0 {
		return 0, fmt.Errorf("error al leer el bloque de datos %d en posición %d: %v", blockNum, blockPosition, err)
	}

	// Determinar cuántos bytes leer del bloque
	bytesToRead := int(structs.FILE_CONTENT_SIZE)
	if bytesToRead > maxBytes {
		bytesToRead = maxBytes
	}

	// Escribir exactamente los bytes necesarios
	blockContent := blockBytes[:bytesToRead]

	fmt.Printf("DEBUG readDirectBlock: bytesToRead=%d, bytesRead=%d, contenido (hex primeros 50): %x\n",
		bytesToRead, bytesRead, blockContent[:min(50, len(blockContent))])
	fmt.Printf("DEBUG readDirectBlock: contenido (string): '%s'\n", string(blockContent))

	content.Write(blockContent)

	return bytesToRead, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
} // readIndirectBlock lee un bloque indirecto simple
func readIndirectBlock(file *os.File, ptrBlockNum int32, blockPos int32, maxBytes int, content *strings.Builder) (int, error) {
	// Leer el bloque de punteros
	blockBytes := make([]byte, structs.BLOCK_SIZE)
	_, err := file.ReadAt(blockBytes, int64(blockPos+(ptrBlockNum*structs.BLOCK_SIZE)))
	if err != nil {
		return 0, err
	}

	var pointerBlock structs.PointerBlock
	buffer := bytes.NewReader(blockBytes)
	binary.Read(buffer, binary.LittleEndian, &pointerBlock)

	totalBytesRead := 0

	// Leer cada bloque apuntado
	for i := 0; i < 16 && pointerBlock.B_pointers[i] != 0 && totalBytesRead < maxBytes; i++ {
		bytesRead, err := readDirectBlock(file, pointerBlock.B_pointers[i], blockPos, maxBytes-totalBytesRead, content)
		if err != nil {
			return totalBytesRead, err
		}
		totalBytesRead += bytesRead
	}

	return totalBytesRead, nil
}

// readDoubleIndirectBlock lee un bloque doblemente indirecto
func readDoubleIndirectBlock(file *os.File, firstLevelBlockNum int32, blockPos int32, maxBytes int, content *strings.Builder) (int, error) {
	// Leer el bloque de primer nivel
	blockBytes := make([]byte, structs.BLOCK_SIZE)
	_, err := file.ReadAt(blockBytes, int64(blockPos+(firstLevelBlockNum*structs.BLOCK_SIZE)))
	if err != nil {
		return 0, err
	}

	var firstLevelPointers structs.PointerBlock
	buffer := bytes.NewReader(blockBytes)
	binary.Read(buffer, binary.LittleEndian, &firstLevelPointers)

	totalBytesRead := 0

	// Para cada entrada en el primer nivel
	for i := 0; i < 16 && firstLevelPointers.B_pointers[i] != 0 && totalBytesRead < maxBytes; i++ {
		bytesRead, err := readIndirectBlock(file, firstLevelPointers.B_pointers[i], blockPos, maxBytes-totalBytesRead, content)
		if err != nil {
			return totalBytesRead, err
		}
		totalBytesRead += bytesRead
	}

	return totalBytesRead, nil
}
