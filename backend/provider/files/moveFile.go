package files

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"main/provider/common"
	"main/provider/mounted"
	"main/structs"
	"main/utils/journaling"
)

func MoveFileOrDirectory(params *structs.MoveParams) error {
	// Verificar que haya una sesión activa
	session := structs.GetSession()
	if !session.Active {
		return fmt.Errorf("debe iniciar sesión para usar el comando MOVE")
	}

	// Obtener la partición montada
	mountManager := mounted.GetMountManager()
	partition, found := mountManager.GetMountedPartition(session.ID)
	if !found {
		return fmt.Errorf("no hay ninguna partición montada para esta sesión")
	}

	// Leer el superbloque
	superblock, err := common.ReadSuperblock(partition.Path, session.ID)
	if err != nil {
		return fmt.Errorf("error leyendo superbloque: %v", err)
	}

	// Leer el MBR del disco
	mbr, err := mounted.ReadMBR(partition.Path)
	if err != nil {
		return fmt.Errorf("error al leer el MBR del disco: %v", err)
	}

	// Buscar la partición montada
	var targetPartition structs.PARTITION
	var partitionFound bool
	for _, p := range mbr.Partitions {
		partName := strings.TrimRight(string(p.Part_name[:]), "\x00")
		if partName == partition.Name {
			targetPartition = p
			partitionFound = true
			break
		}
	}

	if !partitionFound {
		return fmt.Errorf("la partición '%s' no se encontró en el disco", partition.Name)
	}

	// Calcular las posiciones de las estructuras del sistema de archivos
	startPosition := targetPartition.Part_start
	inodeCount := int32(float64(targetPartition.Part_size) * 0.05 / float64(structs.INODE_SIZE))
	blockCount := (targetPartition.Part_size - inodeCount*structs.INODE_SIZE) / structs.BLOCK_SIZE

	bmInodePos := startPosition + structs.SUPERBLOCK_SIZE
	bmBlockPos := bmInodePos + inodeCount
	inodePos := bmBlockPos + blockCount
	blockPos := inodePos + (inodeCount * structs.INODE_SIZE)

	// Abrir el archivo del disco para lectura/escritura
	file, err := os.OpenFile(partition.Path, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo del disco: %v", err)
	}
	defer file.Close()

	// Verificar que el archivo o directorio origen existe
	sourceInodeNum, err := common.FindFileInodeByPath(file, params.Path, superblock, inodePos, blockPos)
	if err != nil {
		return fmt.Errorf("el archivo o directorio de origen no existe: %v", err)
	}

	// Leer el inodo del archivo/directorio origen para verificar permisos
	sourceInodeBytes := make([]byte, structs.INODE_SIZE)
	sourceInodeOffset := int64(superblock.S_inode_start + (sourceInodeNum * structs.INODE_SIZE))
	_, err = file.ReadAt(sourceInodeBytes, sourceInodeOffset)
	if err != nil {
		return fmt.Errorf("error al leer el inodo del origen: %v", err)
	}

	var sourceInode structs.Inode
	buffer := bytes.NewReader(sourceInodeBytes)
	binary.Read(buffer, binary.LittleEndian, &sourceInode)

	// Verificar permisos de escritura sobre el archivo/directorio origen
	if !HasWritePermissionOnDirectory(file, sourceInodeNum, session, inodePos) {
		return errors.New("no tiene permisos de escritura sobre el archivo o directorio origen")
	}

	// Verificar que el directorio de destino existe
	destDir := filepath.Dir(params.Destino)
	destBaseName := filepath.Base(params.Destino)

	// Obtener el nombre original del archivo/directorio a mover
	originalSourceName := filepath.Base(params.Path)

	// Si el destino apunta a un directorio existente, mantener el nombre original
	// Si el destino especifica un nuevo nombre, usar el nombre especificado
	destDirInodeNum, err := common.FindFileInodeByPath(file, params.Destino, superblock, inodePos, blockPos)
	if err == nil {
		// El destino existe - verificar si es un directorio
		destInodeBytes := make([]byte, structs.INODE_SIZE)
		destInodeOffset := int64(superblock.S_inode_start + (destDirInodeNum * structs.INODE_SIZE))
		_, err = file.ReadAt(destInodeBytes, destInodeOffset)
		if err == nil {
			var destInode structs.Inode
			buffer := bytes.NewReader(destInodeBytes)
			binary.Read(buffer, binary.LittleEndian, &destInode)

			if destInode.I_type == 1 { // Es un directorio
				// Mover dentro del directorio manteniendo el nombre original
				destDir = params.Destino
				destBaseName = originalSourceName
				destDirInodeNum, err = common.FindFileInodeByPath(file, destDir, superblock, inodePos, blockPos)
				if err != nil {
					return fmt.Errorf("el directorio de destino no existe: %v", err)
				}
			}
		}
	} else {
		// El destino no existe - usar el directorio padre y el nombre especificado
		destDirInodeNum, err = common.FindFileInodeByPath(file, destDir, superblock, inodePos, blockPos)
		if err != nil {
			return fmt.Errorf("el directorio de destino no existe: %v", err)
		}
	}

	fmt.Printf("DEBUG MOVE: destDir='%s', destBaseName='%s' (calculado)\n", destDir, destBaseName)

	// Leer el inodo del directorio de destino
	destDirInodeBytes := make([]byte, structs.INODE_SIZE)
	destDirInodeOffset := int64(superblock.S_inode_start + (destDirInodeNum * structs.INODE_SIZE))
	_, err = file.ReadAt(destDirInodeBytes, destDirInodeOffset)
	if err != nil {
		return fmt.Errorf("error al leer el inodo del directorio de destino: %v", err)
	}

	var destDirInode structs.Inode
	buffer = bytes.NewReader(destDirInodeBytes)
	binary.Read(buffer, binary.LittleEndian, &destDirInode)

	// Verificar que el destino es un directorio
	if destDirInode.I_type != 1 {
		return errors.New("el destino debe ser un directorio")
	}

	// Verificar permisos de escritura sobre el directorio de destino
	if !HasWritePermissionOnDirectory(file, destDirInodeNum, session, inodePos) {
		return errors.New("no tiene permisos de escritura sobre el directorio de destino")
	}

	// Verificar que no existe ya un archivo/directorio con el mismo nombre en el destino
	fmt.Printf("DEBUG MOVE: Verificando si '%s' ya existe en destino (inodo %d)\n", destBaseName, destDirInodeNum)
	_, existsInDest := FindFileInDirectoryByInode(file, destBaseName, destDirInodeNum, superblock, inodePos, blockPos)
	if existsInDest {
		fmt.Printf("DEBUG MOVE: ERROR - Ya existe '%s' en destino\n", destBaseName)
		return fmt.Errorf("ya existe un archivo o directorio con el nombre '%s' en el destino", destBaseName)
	}
	fmt.Printf("DEBUG MOVE: OK - '%s' no existe en destino, se puede mover\n", destBaseName)

	// Obtener el directorio padre del archivo/directorio origen
	sourceDir := filepath.Dir(params.Path)
	sourceBaseName := filepath.Base(params.Path)
	fmt.Printf("DEBUG MOVE: sourceDir='%s', sourceBaseName='%s'\n", sourceDir, sourceBaseName)

	// Si es el directorio raíz, usar inodo 0
	var sourceDirInodeNum int32 = 0
	if sourceDir != "/" {
		fmt.Printf("DEBUG MOVE: Buscando directorio padre origen: '%s'\n", sourceDir)
		sourceDirInodeNum, err = common.FindFileInodeByPath(file, sourceDir, superblock, inodePos, blockPos)
		if err != nil {
			fmt.Printf("DEBUG MOVE: ERROR - No se pudo encontrar directorio padre: %v\n", err)
			return fmt.Errorf("error al encontrar el directorio padre del origen: %v", err)
		}
		fmt.Printf("DEBUG MOVE: Directorio padre origen encontrado, inodo=%d\n", sourceDirInodeNum)
	}

	// Realizar el movimiento:
	// 1. Agregar entrada en el directorio de destino
	fmt.Printf("DEBUG MOVE: Fase 1 - Agregando '%s' (inodo %d) al directorio destino %d\n", destBaseName, sourceInodeNum, destDirInodeNum)
	err = AddFileToDirectoryWithIndirect(file, destBaseName, sourceInodeNum, destDirInodeNum, superblock, inodePos, blockPos, bmBlockPos, blockCount)
	if err != nil {
		fmt.Printf("DEBUG MOVE: ERROR - Fallo al agregar al destino: %v\n", err)
		return fmt.Errorf("error al agregar entrada en el directorio de destino: %v", err)
	}
	fmt.Printf("DEBUG MOVE: ¡Éxito! Agregado '%s' al directorio destino\n", destBaseName)

	// 2. Eliminar entrada del directorio origen
	fmt.Printf("DEBUG MOVE: Fase 2 - Eliminando '%s' del directorio origen %d\n", sourceBaseName, sourceDirInodeNum)
	err = removeFileFromDirectoryManual(file, sourceDirInodeNum, sourceBaseName, superblock, inodePos, blockPos)
	if err != nil {
		fmt.Printf("DEBUG MOVE: ERROR - Fallo al eliminar del origen: %v\n", err)
		// Si falla, intentar revertir eliminando del destino
		removeFileFromDirectoryManual(file, destDirInodeNum, destBaseName, superblock, inodePos, blockPos)
		return fmt.Errorf("error al eliminar entrada del directorio origen: %v", err)
	}
	fmt.Printf("DEBUG MOVE: ¡Éxito! Eliminado '%s' del directorio origen\n", sourceBaseName)

	// Registrar el movimiento en el journal EXT3
	journaling.LogEXT3Operation(session.ID, "MOVE_FILE_OR_DIRECTORY", params.Path,
		fmt.Sprintf("Archivo/directorio movido de '%s' a '%s'", params.Path, params.Destino))

	return nil
}

// Función auxiliar para remover una entrada de directorio de forma manual
func removeFileFromDirectoryManual(file *os.File, dirInodeNum int32, fileName string, superblock *structs.Superblock, inodePos, blockPos int32) error {
	// Leer el inodo del directorio
	dirInodeBytes := make([]byte, structs.INODE_SIZE)
	dirInodeOffset := int64(superblock.S_inode_start + (dirInodeNum * structs.INODE_SIZE))
	_, err := file.ReadAt(dirInodeBytes, dirInodeOffset)
	if err != nil {
		return fmt.Errorf("error al leer el inodo del directorio: %v", err)
	}

	var dirInode structs.Inode
	buffer := bytes.NewReader(dirInodeBytes)
	binary.Read(buffer, binary.LittleEndian, &dirInode)

	// Buscar la entrada en todos los bloques del directorio
	for i := 0; i < 15; i++ {
		if dirInode.I_block[i] == -1 || dirInode.I_block[i] == 0 {
			continue
		}

		// Leer el bloque
		blockData := make([]byte, structs.BLOCK_SIZE)

		// Calcular la posición del bloque correctamente usando el superblock
		var blockPosition int32
		blockNum := dirInode.I_block[i]
		if superblock.S_block_start == 0 {
			blockPosition = superblock.S_inode_start + (superblock.S_inodes_count * structs.INODE_SIZE) + (blockNum * structs.BLOCK_SIZE)
		} else {
			blockPosition = superblock.S_block_start + (blockNum * structs.BLOCK_SIZE)
		}

		_, err := file.ReadAt(blockData, int64(blockPosition))
		if err != nil {
			continue
		}

		// Interpretar como FolderBlock
		var folderBlock structs.FolderBlock
		folderBuffer := bytes.NewReader(blockData)
		binary.Read(folderBuffer, binary.LittleEndian, &folderBlock)

		// Buscar la entrada en el bloque
		for j := 0; j < 4; j++ {
			entryName := strings.TrimRight(string(folderBlock.B_content[j].B_name[:]), "\x00")
			if entryName == fileName {
				// Limpiar la entrada
				for k := range folderBlock.B_content[j].B_name {
					folderBlock.B_content[j].B_name[k] = 0
				}
				folderBlock.B_content[j].B_inodo = -1

				// Escribir el bloque actualizado
				var buf bytes.Buffer
				binary.Write(&buf, binary.LittleEndian, &folderBlock)
				_, err = file.WriteAt(buf.Bytes(), int64(blockPosition))
				return err
			}
		}
	}

	return fmt.Errorf("entrada '%s' no encontrada en el directorio", fileName)
}
