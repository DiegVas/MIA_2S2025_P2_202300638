package files

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	diskio "main/diskManager"
	"main/provider/common"
	"main/provider/mounted"
	"main/provider/users"
	"main/structs"
	"main/utils/journaling"
	"os"
	"strings"
	"time"
	"unsafe"
)

func FindOrCreateParentDirectory(file *os.File, dirPath string, session *structs.Session, superblock *structs.Superblock,
	inodePos, blockPos, bmInodePos, bmBlockPos, inodeCount, blockCount int32, createParents bool) (int32, error) {

	fmt.Printf("DEBUG FindOrCreateParentDirectory: dirPath='%s', createParents=%t\n", dirPath, createParents)

	if dirPath == "" {
		fmt.Printf("DEBUG FindOrCreateParentDirectory: dirPath vacío, retornando inodo raíz (0)\n")
		return 0, nil
	}

	components := strings.Split(dirPath, "/")
	currentInodeNum := int32(0)

	fmt.Printf("DEBUG FindOrCreateParentDirectory: componentes del path: %v\n", components)

	for i, component := range components {
		if component == "" {
			fmt.Printf("DEBUG FindOrCreateParentDirectory: Saltando componente vacío en índice %d\n", i)
			continue
		}

		fmt.Printf("DEBUG FindOrCreateParentDirectory: Procesando componente %d: '%s', inodo actual: %d\n", i, component, currentInodeNum)

		// ? Buscar el directorio en el inodo actual
		nextInodeNum, exists := FindFileInDirectoryByInode(file, component, currentInodeNum, superblock, inodePos, blockPos)

		if !exists {
			fmt.Printf("DEBUG FindOrCreateParentDirectory: Componente '%s' no existe en inodo %d\n", component, currentInodeNum)

			if !createParents {
				fmt.Printf("DEBUG FindOrCreateParentDirectory: createParents=false, error: directorio '%s' no existe\n", component)
				return -1, fmt.Errorf("el directorio '%s' no existe", component)
			}

			//  ? Obtener la partición montada
			mountManager := mounted.GetMountManager()
			partition, found := mountManager.GetMountedPartition(session.ID)
			if !found {
				fmt.Printf("DEBUG FIND_OR_CREATE: Error: no se encontró partición con ID '%s'\n", session.ID)
				return -1, fmt.Errorf("no se encontró la partición con ID '%s'", session.ID)
			}

			//  ? Crear el directorio
			currentUser, err := users.GetCurrentUserInfo(session, partition)
			if err != nil {
				fmt.Printf("DEBUG FIND_OR_CREATE: Error obteniendo usuario: %v\n", err)
				return -1, fmt.Errorf("error al obtener información del usuario: %v", err)
			}

			nextInodeNum, err = CreateNewDirectory(file, component, currentUser, currentInodeNum, superblock,
				inodePos, blockPos, bmInodePos, bmBlockPos, inodeCount, blockCount)
			if err != nil {
				fmt.Printf("DEBUG FIND_OR_CREATE: Error creando directorio '%s': %v\n", component, err)
				return -1, fmt.Errorf("error al crear directorio '%s': %v", component, err)
			}

			// Construir la ruta completa hasta este directorio
			fullPath := buildPathToComponent(components, i)

			// Registrar la creación del directorio padre en el journal EXT3
			journaling.LogEXT3Operation(session.ID, "CREATE_PARENT_DIRECTORY", fullPath,
				fmt.Sprintf("Directorio padre '%s' creado con permisos 755 (mkdir -p)", component))

		} else {
			fmt.Printf("DEBUG FindOrCreateParentDirectory: Componente '%s' encontrado en inodo %d\n", component, nextInodeNum)
		}

		currentInodeNum = nextInodeNum
		fmt.Printf("DEBUG FindOrCreateParentDirectory: Inodo actual actualizado a: %d\n", currentInodeNum)
	}

	fmt.Printf("DEBUG FindOrCreateParentDirectory: Retornando inodo final: %d\n", currentInodeNum)
	return currentInodeNum, nil
}

func FindFileInDirectoryByInode(file *os.File, name string, dirInodeNum int32, superblock *structs.Superblock, inodePos, blockPos int32) (int32, bool) {
	fmt.Printf("DEBUG FindFileInDirectoryByInode: Buscando '%s' en inodo %d\n", name, dirInodeNum)
	fmt.Printf("DEBUG FindFileInDirectoryByInode: inodePos=%d, blockPos=%d\n", inodePos, blockPos)

	// ? Leer el inodo del directorio
	inodeBytes := make([]byte, structs.INODE_SIZE)
	inodePosition := int64(superblock.S_inode_start + (dirInodeNum * structs.INODE_SIZE))
	fmt.Printf("DEBUG FindFileInDirectoryByInode: Leyendo inodo en posición %d (S_inode_start=%d)\n", inodePosition, superblock.S_inode_start)
	_, err := file.ReadAt(inodeBytes, inodePosition)
	if err != nil {
		fmt.Printf("DEBUG FindFileInDirectoryByInode: Error leyendo inodo: %v\n", err)
		return -1, false
	}

	var inode structs.Inode
	buffer := bytes.NewReader(inodeBytes)
	binary.Read(buffer, binary.LittleEndian, &inode)

	fmt.Printf("DEBUG FindFileInDirectoryByInode: Inodo leído - Type: %d, Size: %d\n", inode.I_type, inode.I_size)
	fmt.Printf("DEBUG FindFileInDirectoryByInode: Bloques asignados: %v\n", inode.I_block[:5]) // Mostrar los primeros 5 bloques

	// ESPECIAL DEBUG PARA INODO RAÍZ
	if dirInodeNum == 0 {
		fmt.Printf("DEBUG INODO RAÍZ: UID=%d, GID=%d, Permisos=%d\n", inode.I_uid, inode.I_gid, inode.I_perm)
		fmt.Printf("DEBUG INODO RAÍZ: Bytes raw del inodo: %v\n", inodeBytes[:40]) // Primeros 40 bytes
	}

	// ? Buscar en todos los bloques directos del directorio
	for i := 0; i < 12; i++ { // Los primeros 12 son bloques directos
		fmt.Printf("DEBUG FindFileInDirectoryByInode: Revisando slot %d: bloque=%d\n", i, inode.I_block[i])

		// Permitir bloque 0 SOLO para el slot 0 del inodo raíz
		// Para todos los demás casos, bloque 0 significa "no asignado"
		blockNum := inode.I_block[i]
		if blockNum > 0 || (dirInodeNum == 0 && i == 0 && blockNum == 0) {
			fmt.Printf("DEBUG FindFileInDirectoryByInode: Procesando bloque %d (blockNum=%d)\n", i, blockNum)

			blockBytes := make([]byte, structs.BLOCK_SIZE)
			// Usar la misma lógica que findFileInDirectory
			var blockPosition int32
			if superblock.S_block_start == 0 {
				// Calcular: inode_start + (inodes_count * inode_size) + (blockNum * block_size)
				blockPosition = superblock.S_inode_start + (superblock.S_inodes_count * structs.INODE_SIZE) + (blockNum * structs.BLOCK_SIZE)
			} else {
				blockPosition = superblock.S_block_start + (blockNum * structs.BLOCK_SIZE)
			}
			fmt.Printf("DEBUG FindFileInDirectoryByInode: Leyendo bloque en posición %d (S_block_start=%d)\n", blockPosition, superblock.S_block_start)
			_, err = file.ReadAt(blockBytes, int64(blockPosition))
			if err != nil {
				fmt.Printf("DEBUG FindFileInDirectoryByInode: Error leyendo bloque %d: %v\n", i, err)
				continue
			}

			var folderBlock structs.FolderBlock
			buffer = bytes.NewReader(blockBytes)
			binary.Read(buffer, binary.LittleEndian, &folderBlock)

			fmt.Printf("DEBUG FindFileInDirectoryByInode: Buscando en bloque %d\n", blockNum)
			// ? Buscar el archivo en las entradas de este bloque
			for j, content := range folderBlock.B_content {
				entryName := strings.TrimRight(string(content.B_name[:]), "\x00")
				fmt.Printf("DEBUG FindFileInDirectoryByInode: Entrada %d: nombre='%s', inodo=%d\n", j, entryName, content.B_inodo)
				if entryName == name {
					fmt.Printf("DEBUG FindFileInDirectoryByInode: ¡Encontrado! '%s' en inodo %d\n", name, content.B_inodo)
					return content.B_inodo, true
				}
			}
		} else {
			fmt.Printf("DEBUG FindFileInDirectoryByInode: Slot %d está vacío (bloque=%d)\n", i, blockNum)
		}
	}

	fmt.Printf("DEBUG FindFileInDirectoryByInode: No se encontró '%s' en inodo %d\n", name, dirInodeNum)
	return -1, false
}

func CreateNewDirectory(file *os.File, dirName string, user *structs.User, parentInodeNum int32, superblock *structs.Superblock,
	inodePos, blockPos, bmInodePos, bmBlockPos, inodeCount, blockCount int32) (int32, error) {

	// Encontrar un inodo libre
	freeInodeNum, err := FindFreeInode(file, bmInodePos, inodeCount)
	if err != nil {
		fmt.Printf("❌ ERROR buscando inodo libre: %v\n", err)
		return -1, err
	}

	// Encontrar un bloque libre

	freeBlocks, err := FindFreeBlocks(file, bmBlockPos, blockCount, 1)
	if err != nil {

		return -1, err
	}
	blockNum := freeBlocks[0]

	// Crear el inodo del directorio
	now := time.Now()
	timeBytes := FormatTimeToBytes(now)

	dirInode := structs.Inode{
		I_uid:   user.UID,
		I_gid:   user.GID,
		I_size:  structs.FOLDER_SIZE,
		I_atime: timeBytes,
		I_ctime: timeBytes,
		I_mtime: timeBytes,
		I_block: [15]int32{blockNum},
		I_type:  1,   // 1 = directorio
		I_perm:  755, // rwxr-xr-x
	}

	// Escribir el inodo
	var inodeBuffer bytes.Buffer
	binary.Write(&inodeBuffer, binary.LittleEndian, dirInode)
	_, err = file.WriteAt(inodeBuffer.Bytes(), int64(inodePos+(freeInodeNum*structs.INODE_SIZE)))
	if err != nil {
		return -1, err
	}

	// Crear el bloque del directorio con entradas . y ..
	folderBlock := structs.FolderBlock{}
	copy(folderBlock.B_content[0].B_name[:], ".")
	folderBlock.B_content[0].B_inodo = freeInodeNum
	copy(folderBlock.B_content[1].B_name[:], "..")
	folderBlock.B_content[1].B_inodo = parentInodeNum

	// Escribir el bloque del directorio
	var folderBuffer bytes.Buffer
	binary.Write(&folderBuffer, binary.LittleEndian, folderBlock)
	_, err = file.WriteAt(folderBuffer.Bytes(), int64(blockPos+(blockNum*structs.BLOCK_SIZE)))
	if err != nil {
		return -1, err
	}

	// Marcar inodo y bloque como usados
	err = MarkInodeAsUsed(file, bmInodePos, freeInodeNum)
	if err != nil {
		return -1, err
	}

	err = MarkBlockAsUsed(file, bmBlockPos, blockNum)
	if err != nil {
		return -1, err
	}

	// Agregar la entrada al directorio padre
	err = AddFileToDirectoryWithIndirect(file, dirName, freeInodeNum, parentInodeNum, superblock, inodePos, blockPos, bmBlockPos, blockCount)
	if err != nil {
		return -1, err
	}

	return freeInodeNum, nil
}

func AddFileToDirectory(file *os.File, fileName string, fileInodeNum, dirInodeNum int32, superblock *structs.Superblock, inodePos, blockPos, bmBlockPos, blockCount int32) error {
	// Usar la nueva implementación con soporte para bloques indirectos
	// Esta función mantiene compatibilidad con código existente
	return AddFileToDirectoryWithIndirect(file, fileName, fileInodeNum, dirInodeNum, superblock, inodePos, blockPos, bmBlockPos, blockCount)
}

// AddFileToDirectoryLegacy es la implementación original (solo bloques directos)
// Se mantiene para referencia y casos especiales
func AddFileToDirectoryLegacy(file *os.File, fileName string, fileInodeNum, dirInodeNum, inodePos, blockPos, bmBlockPos, blockCount int32) error {
	// Leer el inodo del directorio
	inodeBytes := make([]byte, structs.INODE_SIZE)
	_, err := file.ReadAt(inodeBytes, int64(inodePos+(dirInodeNum*structs.INODE_SIZE)))
	if err != nil {
		return err
	}

	var dirInode structs.Inode
	buffer := bytes.NewReader(inodeBytes)
	binary.Read(buffer, binary.LittleEndian, &dirInode)

	// Buscar en todos los bloques directos existentes
	for i := 0; i < 12; i++ { // Los primeros 12 son bloques directos
		if dirInode.I_block[i] != 0 {
			// Verificar si hay espacio en este bloque
			blockNum := dirInode.I_block[i]
			blockBytes := make([]byte, structs.BLOCK_SIZE)
			_, err = file.ReadAt(blockBytes, int64(blockPos+(blockNum*structs.BLOCK_SIZE)))
			if err != nil {
				continue
			}

			var folderBlock structs.FolderBlock
			buffer = bytes.NewReader(blockBytes)
			binary.Read(buffer, binary.LittleEndian, &folderBlock)

			// Buscar una entrada libre en este bloque
			for j := range folderBlock.B_content {
				entryName := strings.TrimRight(string(folderBlock.B_content[j].B_name[:]), "\x00")
				if entryName == "" {
					// Entrada libre encontrada
					copy(folderBlock.B_content[j].B_name[:], fileName)
					folderBlock.B_content[j].B_inodo = fileInodeNum

					// Escribir el bloque actualizado
					var folderBuffer bytes.Buffer
					binary.Write(&folderBuffer, binary.LittleEndian, folderBlock)
					_, err = file.WriteAt(folderBuffer.Bytes(), int64(blockPos+(blockNum*structs.BLOCK_SIZE)))
					return err
				}
			}
		} else {
			// Este slot está libre, crear un nuevo bloque de carpeta
			newBlocks, err := FindFreeBlocks(file, bmBlockPos, blockCount, 1)
			if err != nil {
				// Si no hay bloques libres, retornar error específico
				return fmt.Errorf("no hay bloques libres disponibles para expandir el directorio: %v", err)
			}

			newBlockNum := newBlocks[0]
			dirInode.I_block[i] = newBlockNum

			// Crear nuevo FolderBlock
			var newFolderBlock structs.FolderBlock
			copy(newFolderBlock.B_content[0].B_name[:], fileName)
			newFolderBlock.B_content[0].B_inodo = fileInodeNum

			// Escribir el nuevo bloque
			var folderBuffer bytes.Buffer
			binary.Write(&folderBuffer, binary.LittleEndian, newFolderBlock)
			_, err = file.WriteAt(folderBuffer.Bytes(), int64(blockPos+(newBlockNum*structs.BLOCK_SIZE)))
			if err != nil {
				return err
			}

			// Marcar el bloque como usado
			err = MarkBlockAsUsed(file, bmBlockPos, newBlockNum)
			if err != nil {
				return err
			}

			// Actualizar el inodo del directorio
			var inodeBuffer bytes.Buffer
			binary.Write(&inodeBuffer, binary.LittleEndian, dirInode)
			_, err = file.WriteAt(inodeBuffer.Bytes(), int64(inodePos+(dirInodeNum*structs.INODE_SIZE)))
			return err
		}
	}

	return errors.New("no hay espacio disponible en el directorio padre - todos los bloques directos están llenos")
}

// CreateDirectory crea un directorio usando el comando mkdir
func CreateDirectory(params *structs.MkdirParams) error {
	session := structs.GetSession()
	if !session.Active {
		return errors.New("debe iniciar sesión para usar el comando MKDIR")
	}

	mountManager := mounted.GetMountManager()
	partition, found := mountManager.GetMountedPartition(session.ID)
	if !found {
		return errors.New("no hay ninguna partición montada para esta sesión")
	}

	mbr, err := diskio.ReadMBR(partition.Path)
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

	// Abrir el archivo del disco para escritura
	diskFile, err := os.OpenFile(partition.Path, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error al abrir el disco: %v", err)
	}
	defer diskFile.Close()

	// Detectar el tipo de sistema de archivos leyendo el superbloque
	startPosition := targetPartition.Part_start
	realSuperbSize := int(unsafe.Sizeof(structs.Superblock{}))
	superbBytes := make([]byte, realSuperbSize)
	_, err = diskFile.ReadAt(superbBytes, int64(startPosition))
	if err != nil {
		return fmt.Errorf("error al leer el superbloque: %v", err)
	}

	// Deserializar el superbloque para obtener el tipo de sistema de archivos
	var superblock structs.Superblock
	superbBuffer := bytes.NewReader(superbBytes)
	err = binary.Read(superbBuffer, binary.LittleEndian, &superblock)
	if err != nil {
		return fmt.Errorf("error al deserializar el superbloque: %v", err)
	}

	// Verificar si la partición está formateada
	if superblock.S_filesystem_type == 0 {
		return fmt.Errorf("la partición '%s' no está formateada. Use el comando 'mkfs' para formatearla primero", partition.Name)
	}

	// Verificar que sea un sistema de archivos válido
	if superblock.S_filesystem_type != 2 && superblock.S_filesystem_type != 3 {
		return fmt.Errorf("tipo de sistema de archivos no válido: %d. Solo se soportan EXT2 (2) y EXT3 (3)", superblock.S_filesystem_type)
	}

	// Usar las posiciones almacenadas en el superbloque
	bmInodePos := superblock.S_bm_inode_start
	bmBlockPos := superblock.S_bm_block_start
	inodePos := superblock.S_inode_start
	blockPos := superblock.S_block_start
	inodeCount := superblock.S_inodes_count
	blockCount := superblock.S_blocks_count

	fmt.Printf("DEBUG CreateDirectory: filesystem_type=%d, usando posiciones del superbloque\n", superblock.S_filesystem_type)
	fmt.Printf("DEBUG CreateDirectory: inodePos=%d, blockPos=%d\n", inodePos, blockPos)

	// Si el parámetro -p está presente, crear directorios padre
	if params.P {
		_, err = FindOrCreateParentDirectory(diskFile, params.Path, session, &superblock,
			inodePos, blockPos, bmInodePos, bmBlockPos, inodeCount, blockCount, true)
		if err != nil {
			return fmt.Errorf("error al crear directorios padre: %v", err)
		}

		// Registrar la operación final (directorio objetivo) en el journal EXT3
		journaling.LogEXT3Operation(session.ID, "CREATE_DIRECTORY_WITH_PARENTS", params.Path,
			fmt.Sprintf("Directorio objetivo '%s' creado con mkdir -p (incluyendo padres)", params.Path))

		return nil
	}

	// Sin -p, verificar que el directorio padre existe y crear solo el directorio especificado
	parentPath := GetParentPath(params.Path)
	dirName := GetLastComponent(params.Path)

	// Buscar el inodo del directorio padre
	parentInodeNum, err := common.FindFileInodeByPath(diskFile, parentPath, &superblock, inodePos, blockPos)
	if err != nil {
		fmt.Printf("DEBUG MKDIR: Error buscando directorio padre: %v\n", err)
		return fmt.Errorf("el directorio padre no existe: %v", err)
	}

	// Crear solo el directorio especificado
	currentUser, err := users.GetCurrentUserInfo(session, partition)
	if err != nil {
		return fmt.Errorf("error al obtener información del usuario: %v", err)
	}

	_, err = CreateNewDirectory(diskFile, dirName, currentUser, parentInodeNum, &superblock,
		inodePos, blockPos, bmInodePos, bmBlockPos, inodeCount, blockCount)
	if err != nil {
		return fmt.Errorf("error al crear el directorio: %v", err)
	}

	// Registrar la operación en el journal EXT3
	journaling.LogEXT3Operation(session.ID, "CREATE_DIRECTORY", params.Path,
		fmt.Sprintf("Directorio '%s' creado sin -p con permisos 755", dirName))

	return nil
}

// getParentPath obtiene la ruta del directorio padre
func GetParentPath(path string) string {

	if path == "/" {

		return "/"
	}

	path = strings.TrimSuffix(path, "/")

	lastSlash := strings.LastIndex(path, "/")

	if lastSlash == 0 {

		return "/"
	}
	if lastSlash == -1 {

		return "/"
	}

	result := path[:lastSlash]

	return result
}

// getLastComponent extrae el último componente de una ruta
func GetLastComponent(path string) string {

	path = strings.TrimSuffix(path, "/")

	parts := strings.Split(path, "/")

	if len(parts) == 0 {
		fmt.Printf("DEBUG getLastComponent: No hay parts, retornando ''\n")
		return ""
	}

	result := parts[len(parts)-1]
	return result
}

// buildPathToComponent construye la ruta completa hasta el componente especificado
func buildPathToComponent(components []string, index int) string {
	if index < 0 || index >= len(components) {
		return "/"
	}

	var pathParts []string
	for i := 0; i <= index; i++ {
		if components[i] != "" {
			pathParts = append(pathParts, components[i])
		}
	}

	if len(pathParts) == 0 {
		return "/"
	}

	return "/" + strings.Join(pathParts, "/")
}
