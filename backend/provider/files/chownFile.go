package files

import (
	"encoding/binary"
	"errors"
	"fmt"
	"main/provider/common"
	"main/provider/mounted"
	"main/provider/reports"
	"main/provider/users"
	"main/structs"
	"os"
	"strings"
)

// ChangeOwner cambia el propietario de archivos y carpetas
func ChangeOwner(params *structs.ChownParams) error {
	// Verificar que haya una sesión activa
	session := structs.GetSession()
	if !session.Active {
		return errors.New("debe iniciar sesión para usar el comando CHOWN")
	}

	// Solo root puede cambiar propietarios (a menos que sea el propio archivo del usuario)
	if session.User != "root" {
		return errors.New("solo el usuario root puede cambiar el propietario de archivos")
	}

	// Obtener la partición montada
	mountManager := mounted.GetMountManager()
	partition, found := mountManager.GetMountedPartition(session.ID)
	if !found {
		return errors.New("no hay ninguna partición montada para esta sesión")
	}

	// Leer el superbloque
	superblock, err := common.ReadSuperblock(partition.Path, session.ID)
	if err != nil {
		return fmt.Errorf("error leyendo superbloque: %v", err)
	}

	// Calcular posiciones como en moveFile.go
	mbr, err := mounted.ReadMBR(partition.Path)
	if err != nil {
		return fmt.Errorf("error al leer el MBR del disco: %v", err)
	}

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

	// Abrir el archivo del disco
	file, err := os.OpenFile(partition.Path, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error abriendo disco: %v", err)
	}
	defer file.Close()

	// Verificar que el usuario nuevo existe
	userExists, newUserUID, err := verifyUserExists(file, superblock, params.Usuario, partition)
	if err != nil {
		return fmt.Errorf("error verificando usuario: %v", err)
	}
	if !userExists {
		return fmt.Errorf("el usuario '%s' no existe", params.Usuario)
	}

	// Buscar el archivo/carpeta usando la función que funciona en moveFile
	inodeNum, err := common.FindFileInodeByPath(file, params.Path, inodePos, blockPos)
	if err != nil {
		return fmt.Errorf("no se encontró la ruta '%s': %v", params.Path, err)
	}

	// Verificar permisos
	err = verifyChownPermissions(file, superblock, inodeNum, session, partition)
	if err != nil {
		return err
	}

	// Cambiar el propietario
	err = changeOwnerRecursive(file, superblock, inodeNum, newUserUID, params.R)
	if err != nil {
		return fmt.Errorf("error cambiando propietario: %v", err)
	}

	return nil
}

// verifyUserExists verifica si el usuario existe y retorna su UID
func verifyUserExists(file *os.File, superblock *structs.Superblock, username string, partition mounted.MountedPartition) (bool, int32, error) {
	// Usar la función existente para leer el archivo de usuarios
	usersFile, err := users.ReadUsersFile(partition.Path, partition.Name)
	if err != nil {
		return false, 0, fmt.Errorf("error al leer el archivo de usuarios: %v", err)
	}

	// Buscar el usuario en el contenido
	lines := strings.Split(usersFile, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Split(line, ",")
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}

		// Formato: UID, U, grupo, usuario, contraseña
		if len(fields) >= 4 && fields[1] == "U" && fields[3] == username && fields[0] != "0" {
			// Usuario encontrado, convertir UID
			var uid int32
			fmt.Sscanf(fields[0], "%d", &uid)
			return true, uid, nil
		}
	}

	return false, 0, nil
}

// verifyChownPermissions verifica si el usuario tiene permisos para cambiar el propietario
func verifyChownPermissions(file *os.File, superblock *structs.Superblock, inodeNum int32, session *structs.Session, partition mounted.MountedPartition) error {
	// Root puede cambiar cualquier propietario
	if session.User == "root" {
		return nil
	}

	// Si no es root, no puede cambiar propietarios (ya se verificó arriba, pero por seguridad)
	return errors.New("solo el usuario root puede cambiar el propietario de archivos")
}

// changeOwnerRecursive cambia el propietario del archivo/carpeta especificado
func changeOwnerRecursive(file *os.File, superblock *structs.Superblock, inodeNum int32, newUID int32, recursive bool) error {
	// Leer el inodo
	inode, err := reports.ReadInodeAt(file, inodeNum, superblock.S_inode_start)
	if err != nil {
		return fmt.Errorf("error leyendo inodo: %v", err)
	}

	// Cambiar el propietario del inodo actual
	inode.I_uid = newUID

	// Escribir el inodo actualizado
	err = writeInodeAt(file, inodeNum, superblock.S_inode_start, inode)
	if err != nil {
		return fmt.Errorf("error escribiendo inodo: %v", err)
	}

	// Si es una carpeta y se especificó recursivo, cambiar también el contenido
	if inode.I_type == 1 && recursive {
		err = changeOwnerInDirectory(file, superblock, inode, newUID)
		if err != nil {
			return fmt.Errorf("error cambiando propietario recursivamente: %v", err)
		}
	}

	return nil
}

// changeOwnerInDirectory cambia el propietario de todos los archivos en una carpeta recursivamente
func changeOwnerInDirectory(file *os.File, superblock *structs.Superblock, dirInode *structs.Inode, newUID int32) error {
	// Recorrer todos los bloques de la carpeta
	for i := 0; i < 15; i++ {
		if dirInode.I_block[i] == -1 {
			continue
		}

		blockPos := int64(superblock.S_block_start + (dirInode.I_block[i] * structs.BLOCK_SIZE))

		// Leer el bloque de carpeta
		var folderBlock structs.FolderBlock
		_, err := file.Seek(blockPos, 0)
		if err != nil {
			continue
		}

		err = binary.Read(file, binary.LittleEndian, &folderBlock)
		if err != nil {
			continue
		}

		// Procesar cada entrada en el bloque
		for j := 0; j < 4; j++ {
			entry := folderBlock.B_content[j]

			// Saltar entradas vacías, "." y ".."
			entryName := strings.TrimRight(string(entry.B_name[:]), "\x00")
			if entry.B_inodo == -1 || entryName == "" || entryName == "." || entryName == ".." {
				continue
			}

			// Cambiar propietario recursivamente
			err = changeOwnerRecursive(file, superblock, entry.B_inodo, newUID, true)
			if err != nil {
				// Continuar con otros archivos aunque uno falle
				continue
			}
		}
	}

	return nil
}

// writeInodeAt escribe un inodo en la posición especificada
func writeInodeAt(file *os.File, inodeNum int32, inodeStart int32, inode *structs.Inode) error {
	inodePos := int64(inodeStart + (inodeNum * structs.INODE_SIZE))

	_, err := file.Seek(inodePos, 0)
	if err != nil {
		return fmt.Errorf("error posicionando archivo: %v", err)
	}

	err = binary.Write(file, binary.LittleEndian, inode)
	if err != nil {
		return fmt.Errorf("error escribiendo inodo: %v", err)
	}

	return nil
}

// Funciones auxiliares que pueden ya existir en otros archivos
// Si existen, estas definiciones se pueden eliminar

// findInodeByPath busca un inodo por su ruta
func findInodeByPath(file *os.File, superblock *structs.Superblock, path string) (int32, error) {
	// Normalizar la ruta
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Comenzar desde el directorio raíz (inodo 0)
	currentInode := int32(0)

	// Si la ruta es solo "/", retornar el inodo raíz
	if path == "/" {
		return currentInode, nil
	}

	// Dividir la ruta en componentes
	components := strings.Split(strings.Trim(path, "/"), "/")

	// Buscar cada componente
	for _, component := range components {
		if component == "" {
			continue
		}

		nextInode, err := findInodeInDirectory(file, superblock, currentInode, component)
		if err != nil {
			return -1, err
		}
		currentInode = nextInode
	}

	return currentInode, nil
}

// findInodeInDirectory busca un archivo/carpeta específico dentro de una carpeta
func findInodeInDirectory(file *os.File, superblock *structs.Superblock, dirInodeNum int32, name string) (int32, error) {
	// Leer el inodo del directorio
	dirInode, err := reports.ReadInodeAt(file, dirInodeNum, superblock.S_inode_start)
	if err != nil {
		return -1, fmt.Errorf("error leyendo inodo del directorio: %v", err)
	}

	// Verificar que es un directorio
	if dirInode.I_type != 1 {
		return -1, errors.New("no es un directorio")
	}

	// Buscar en todos los bloques del directorio
	for i := 0; i < 15; i++ {
		if dirInode.I_block[i] == -1 {
			continue
		}

		blockPos := int64(superblock.S_block_start + (dirInode.I_block[i] * structs.BLOCK_SIZE))

		// Leer el bloque de carpeta
		var folderBlock structs.FolderBlock
		_, err := file.Seek(blockPos, 0)
		if err != nil {
			continue
		}

		err = binary.Read(file, binary.LittleEndian, &folderBlock)
		if err != nil {
			continue
		}

		// Buscar el nombre en las entradas del bloque
		for j := 0; j < 4; j++ {
			entry := folderBlock.B_content[j]
			entryName := strings.TrimRight(string(entry.B_name[:]), "\x00")

			if entry.B_inodo != -1 && entryName == name {
				return entry.B_inodo, nil
			}
		}
	}

	return -1, fmt.Errorf("no se encontró '%s'", name)
}

// findFileRecursively busca un archivo recursivamente en el sistema de archivos
func findFileRecursively(file *os.File, filename string, currentInodeNum int32, superblock *structs.Superblock) (int32, error) {
	// Leer el inodo actual
	currentInode, err := reports.ReadInodeAt(file, currentInodeNum, superblock.S_inode_start)
	if err != nil {
		return -1, fmt.Errorf("error leyendo inodo: %v", err)
	}

	// Si es un archivo, verificar si es el que buscamos
	if currentInode.I_type == 0 {
		return -1, fmt.Errorf("no es un directorio")
	}

	// Si es un directorio, buscar en sus entradas
	for i := 0; i < 15; i++ {
		if currentInode.I_block[i] == -1 {
			continue
		}

		blockPos := int64(superblock.S_block_start + (currentInode.I_block[i] * structs.BLOCK_SIZE))

		// Leer el bloque de carpeta
		var folderBlock structs.FolderBlock
		_, err := file.Seek(blockPos, 0)
		if err != nil {
			continue
		}

		err = binary.Read(file, binary.LittleEndian, &folderBlock)
		if err != nil {
			continue
		}

		// Revisar cada entrada
		for j := 0; j < 4; j++ {
			entry := folderBlock.B_content[j]
			entryName := strings.TrimRight(string(entry.B_name[:]), "\x00")

			if entry.B_inodo == -1 || entryName == "" || entryName == "." || entryName == ".." {
				continue
			}

			if entryName == filename {
				return entry.B_inodo, nil
			}

			// Si es un directorio, buscar recursivamente
			entryInode, err := reports.ReadInodeAt(file, entry.B_inodo, superblock.S_inode_start)
			if err != nil {
				continue
			}

			if entryInode.I_type == 1 {
				result, err := findFileRecursively(file, filename, entry.B_inodo, superblock)
				if err == nil {
					return result, nil
				}
			}
		}
	}

	return -1, fmt.Errorf("archivo no encontrado")
}

// readFileContent lee el contenido completo de un archivo
func readFileContent(file *os.File, inode *structs.Inode, superblock *structs.Superblock) (string, error) {
	var content strings.Builder

	// Leer todos los bloques del archivo
	for i := 0; i < 15; i++ {
		if inode.I_block[i] == -1 {
			break
		}

		blockPos := int64(superblock.S_block_start + (inode.I_block[i] * structs.BLOCK_SIZE))

		// Leer el bloque
		blockData := make([]byte, 64) // Solo los primeros 64 bytes son contenido útil
		_, err := file.ReadAt(blockData, blockPos)
		if err != nil {
			return "", fmt.Errorf("error leyendo bloque: %v", err)
		}

		// Agregar el contenido del bloque
		content.Write(blockData)
	}

	return content.String(), nil
}
