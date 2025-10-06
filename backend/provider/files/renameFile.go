package files

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"main/provider/common"
	"main/provider/mounted"
	"main/provider/reports"
	"main/provider/users"
	"main/structs"
)

func RenameFile(params *structs.RenameFileParams) error {
	// Verificar que haya una sesión activa
	session := structs.GetSession()
	if !session.Active {
		return fmt.Errorf("debe iniciar sesión para usar el comando RENAME")
	}

	// Obtener la partición montada
	mountManager := mounted.GetMountManager()
	partition, found := mountManager.GetMountedPartition(session.ID)
	if !found {
		return fmt.Errorf("no hay ninguna partición montada para esta sesión")
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

	// Abrir el archivo del disco para escritura
	file, err := os.OpenFile(partition.Path, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo del disco: %v", err)
	}
	defer file.Close()

	inodeNum, err := common.FindFileInodeByPath(file, params.Path, inodePos, blockPos)
	if err != nil {
		return fmt.Errorf("error al encontrar el inode del archivo: %v", err)
	}

	// Leer el inodo del archivo
	inodeBytes := make([]byte, structs.INODE_SIZE)
	_, err = file.ReadAt(inodeBytes, int64(inodePos+(inodeNum*structs.INODE_SIZE)))
	if err != nil {
		return fmt.Errorf("error al leer el inodo del archivo: %v", err)
	}

	// Deserializar el inodo
	var inode structs.Inode
	buffer := bytes.NewReader(inodeBytes)
	binary.Read(buffer, binary.LittleEndian, &inode)

	// Verificar que es un archivo (no un directorio)
	if inode.I_type != 0 {
		return errors.New("el path especificado no es un archivo")
	}

	// Obtener información del usuario actual
	currentUser, err := users.GetCurrentUserInfo(session, partition)
	if err != nil {
		return fmt.Errorf("error al obtener información del usuario: %v", err)
	}

	// Verificar permisos de lectura y escritura
	if !hasReadAndWritePermission(inode, currentUser) {
		return errors.New("no tiene permisos de lectura y escritura para este archivo")
	}

	now := time.Now()
	inode.I_mtime = FormatTimeToBytes(now)

	// Leer el superbloque
	superblock, err := common.ReadSuperblock(partition.Path, partition.ID)
	if err != nil {
		return fmt.Errorf("error leyendo superbloque: %v", err)
	}

	// Obtener el directorio padre
	parentPath := GetParentPath(params.Path)
	fileName := GetLastComponent(params.Path)

	parentInodeNum, err := common.FindFileInodeByPath(file, parentPath, inodePos, blockPos)
	if err != nil {
		return fmt.Errorf("error al encontrar el inode del directorio padre: %v", err)
	}

	dirInode, err := reports.ReadInodeAt(file, parentInodeNum, superblock.S_inode_start)
	if err != nil {
		return fmt.Errorf("error al leer el inodo del directorio padre: %v", err)
	}

	// Verificar que sea un directorio
	if dirInode.I_type != 1 {
		return fmt.Errorf("el path padre '%s' no es un directorio", parentPath)
	}

	// Obtener entradas del directorio
	entries, err := reports.GetDirectoryEntries(file, dirInode, superblock, parentInodeNum)
	if err != nil {
		return fmt.Errorf("error al obtener las entradas del directorio: %v", err)
	}

	fmt.Printf("DEBUG - Entradas en el directorio padre: %+v\n", entries)

	// Verificar si ya existe un archivo con el nuevo nombre
	for _, entry := range entries {
		if entry.Name == params.NewName {
			return fmt.Errorf("ya existe un archivo o directorio con el nombre '%s' en el directorio padre", params.NewName)
		}
	}

	// Buscar y actualizar el nombre del archivo en los bloques del directorio padre
	updated := false
	for i := 0; i < 12; i++ { // Bloques directos del directorio padre
		if dirInode.I_block[i] != 0 {
			// Leer el bloque de carpeta
			blockBytes := make([]byte, structs.BLOCK_SIZE)

			var blockPos int32
			if superblock.S_block_start == 0 {
				blockPos = superblock.S_inode_start + (superblock.S_inodes_count * structs.INODE_SIZE)
				blockPos += (dirInode.I_block[i] * structs.BLOCK_SIZE)
			} else {
				blockPos = superblock.S_block_start + (dirInode.I_block[i] * structs.BLOCK_SIZE)
			}

			_, err = file.ReadAt(blockBytes, int64(blockPos))
			if err != nil {
				continue
			}

			// Deserializar el FolderBlock
			var folderBlock structs.FolderBlock
			buffer := bytes.NewReader(blockBytes)
			err = binary.Read(buffer, binary.LittleEndian, &folderBlock)
			if err != nil {
				continue
			}

			// Buscar la entrada del archivo en el bloque
			for j := 0; j < 4; j++ {
				entryName := strings.TrimRight(string(folderBlock.B_content[j].B_name[:]), "\x00")
				if entryName == fileName && folderBlock.B_content[j].B_inodo == inodeNum {
					// Actualizar el nombre
					copy(folderBlock.B_content[j].B_name[:], []byte(params.NewName))
					// Llenar con ceros el resto si el nuevo nombre es más corto
					if len(params.NewName) < 12 {
						for k := len(params.NewName); k < 12; k++ {
							folderBlock.B_content[j].B_name[k] = 0
						}
					}

					// Serializar y escribir el bloque actualizado
					var bufferWrite bytes.Buffer
					err = binary.Write(&bufferWrite, binary.LittleEndian, &folderBlock)
					if err != nil {
						return fmt.Errorf("error al serializar el bloque de carpeta: %v", err)
					}

					_, err = file.WriteAt(bufferWrite.Bytes(), int64(blockPos))
					if err != nil {
						return fmt.Errorf("error al escribir el bloque de carpeta actualizado: %v", err)
					}

					updated = true
					fmt.Printf("Archivo renombrado exitosamente de '%s' a '%s'\n", fileName, params.NewName)
					break
				}
			}

			if updated {
				break
			}
		}
	}

	if !updated {
		return fmt.Errorf("no se pudo encontrar la entrada del archivo '%s' en el directorio padre para actualizar", fileName)
	}

	return nil
}
