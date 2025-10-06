package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"main/structs"
	"os"
	"unsafe"
)

// FilesystemPositions contiene las posiciones calculadas del sistema de archivos
type FilesystemPositions struct {
	FilesystemType int32
	BmInodePos     int32
	BmBlockPos     int32
	InodePos       int32
	BlockPos       int32
	InodeCount     int32
	BlockCount     int32
}

// GetFilesystemPositions lee el superbloque y devuelve las posiciones del sistema de archivos
func GetFilesystemPositions(file *os.File, startPosition int32) (*FilesystemPositions, error) {
	// Leer el superbloque
	realSuperbSize := int(unsafe.Sizeof(structs.Superblock{}))
	superbBytes := make([]byte, realSuperbSize)
	_, err := file.ReadAt(superbBytes, int64(startPosition))
	if err != nil {
		return nil, fmt.Errorf("error al leer el superbloque: %v", err)
	}

	// Deserializar el superbloque
	var superblock structs.Superblock
	superbBuffer := bytes.NewReader(superbBytes)
	err = binary.Read(superbBuffer, binary.LittleEndian, &superblock)
	if err != nil {
		return nil, fmt.Errorf("error al deserializar el superbloque: %v", err)
	}

	// Verificar si la partición está formateada
	if superblock.S_filesystem_type == 0 {
		return nil, fmt.Errorf("la partición no está formateada. Use el comando 'mkfs' para formatearla primero")
	}

	// Verificar que sea un sistema de archivos válido
	if superblock.S_filesystem_type != 2 && superblock.S_filesystem_type != 3 {
		return nil, fmt.Errorf("tipo de sistema de archivos no válido: %d. Solo se soportan EXT2 (2) y EXT3 (3)", superblock.S_filesystem_type)
	}

	// Devolver las posiciones del superbloque
	return &FilesystemPositions{
		FilesystemType: superblock.S_filesystem_type,
		BmInodePos:     superblock.S_bm_inode_start,
		BmBlockPos:     superblock.S_bm_block_start,
		InodePos:       superblock.S_inode_start,
		BlockPos:       superblock.S_block_start,
		InodeCount:     superblock.S_inodes_count,
		BlockCount:     superblock.S_blocks_count,
	}, nil
}
