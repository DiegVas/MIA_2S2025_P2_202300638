package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"main/structs"
	"os"
	"strings"
)

func findFileInDirectory(file *os.File, name string, dirInode structs.Inode, superblock *structs.Superblock, blockPos int32) (int32, error) {
	fmt.Printf("DEBUG findFileInDirectory: Buscando '%s', dirInode.I_size=%d, blockPos=%d\n", name, dirInode.I_size, blockPos)

	// Calcular cuántos bloques necesita este directorio basándose en su tamaño
	if dirInode.I_size <= 0 {
		return -1, fmt.Errorf("directorio vacío")
	}

	// Buscar en todos los bloques directos válidos (no solo los necesarios según tamaño)
	for i := int32(0); i < 12; i++ { // Revisar todos los 12 bloques directos posibles
		blockNum := dirInode.I_block[i]
		fmt.Printf("DEBUG findFileInDirectory: Revisando bloque %d, blockNum=%d\n", i, blockNum)

		// Nota: El bloque 0 SÍ puede ser válido, solo omitimos si es -1 o similar
		if blockNum >= 0 { // Verificar que el bloque sea válido (>= 0)

			blockBytes := make([]byte, structs.BLOCK_SIZE)

			// Usar la misma lógica que GetDirectoryEntries para calcular la posición del bloque
			var blockPosition int32
			if superblock.S_block_start == 0 {
				// Calcular: inode_start + (inodes_count * inode_size) + (blockNum * block_size)
				blockPosition = superblock.S_inode_start + (superblock.S_inodes_count * structs.INODE_SIZE) + (blockNum * structs.BLOCK_SIZE)
			} else {
				blockPosition = superblock.S_block_start + (blockNum * structs.BLOCK_SIZE)
			}

			fmt.Printf("DEBUG findFileInDirectory: Leyendo bloque %d en posición %d (S_block_start=%d)\n", blockNum, blockPosition, superblock.S_block_start)

			_, err := file.ReadAt(blockBytes, int64(blockPosition))
			if err != nil {
				fmt.Printf("DEBUG findFileInDirectory: Error leyendo bloque: %v\n", err)
				continue
			}

			// Deserializar el bloque de directorio
			var folderBlock structs.FolderBlock
			buffer := bytes.NewReader(blockBytes)
			binary.Read(buffer, binary.LittleEndian, &folderBlock)

			// Buscar el archivo en las entradas de este bloque
			for j, content := range folderBlock.B_content {
				entryName := strings.TrimRight(string(content.B_name[:]), "\x00")
				fmt.Printf("DEBUG findFileInDirectory: Entrada %d: name='%s', inodo=%d\n", j, entryName, content.B_inodo)

				if entryName == name {
					fmt.Printf("DEBUG findFileInDirectory: ¡Encontrado! '%s' en inodo %d\n", name, content.B_inodo)
					return content.B_inodo, nil
				}
			}
		} // Cerrar el if blockNum >= 0
	}

	fmt.Printf("DEBUG findFileInDirectory: NO encontrado '%s'\n", name)
	return -1, fmt.Errorf("archivo o directorio '%s' no encontrado", name)
}
