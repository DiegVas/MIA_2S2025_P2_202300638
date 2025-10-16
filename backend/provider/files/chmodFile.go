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
	"main/utils/journaling"
	"os"
	"strconv"
	"strings"
)

// ChangePermissions cambia los permisos de archivos y carpetas
func ChangePermissions(params *structs.ChmodParams) error {
	// Verificar que haya una sesión activa
	session := structs.GetSession()
	if !session.Active {
		return errors.New("debe iniciar sesión para usar el comando CHMOD")
	}

	// Solo root puede usar chmod
	if session.User != "root" {
		return errors.New("solo el usuario root puede cambiar permisos de archivos")
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

	// Convertir UGO string a permisos octales
	permissions, err := ugoToOctal(params.UGO)
	if err != nil {
		return fmt.Errorf("error convirtiendo permisos: %v", err)
	}

	// Buscar el archivo/carpeta usando la función que funciona en moveFile
	inodeNum, err := common.FindFileInodeByPath(file, params.Path, superblock, inodePos, blockPos)
	if err != nil {
		return fmt.Errorf("no se encontró la ruta '%s': %v", params.Path, err)
	}

	// Si no es recursivo o no es root, verificar propiedad
	if !params.R && session.User != "root" {
		err = verifyOwnership(file, superblock, inodeNum, session, partition)
		if err != nil {
			return err
		}
	}

	// Cambiar los permisos
	err = changePermissionsRecursive(file, superblock, inodeNum, permissions, params.R, session, partition)
	if err != nil {
		return fmt.Errorf("error cambiando permisos: %v", err)
	}

	// Registrar el cambio de permisos en el journal EXT3
	recursiveStr := ""
	if params.R {
		recursiveStr = " (recursivo)"
	}
	journaling.LogEXT3Operation(session.ID, "CHANGE_PERMISSIONS", params.Path,
		fmt.Sprintf("Permisos cambiados a '%s'%s", params.UGO, recursiveStr))

	return nil
}

// ugoToOctal convierte el string UGO (ej: "764") a permisos octales
func ugoToOctal(ugo string) (int32, error) {
	if len(ugo) != 3 {
		return 0, errors.New("UGO debe tener exactamente 3 dígitos")
	}

	// Convertir cada dígito
	u, err := strconv.Atoi(string(ugo[0]))
	if err != nil {
		return 0, fmt.Errorf("dígito de usuario inválido: %v", err)
	}

	g, err := strconv.Atoi(string(ugo[1]))
	if err != nil {
		return 0, fmt.Errorf("dígito de grupo inválido: %v", err)
	}

	o, err := strconv.Atoi(string(ugo[2]))
	if err != nil {
		return 0, fmt.Errorf("dígito de otros inválido: %v", err)
	}

	// Combinar en formato octal
	permissions := int32(u*64 + g*8 + o) // 64 = 8^2, 8 = 8^1, 1 = 8^0
	return permissions, nil
}

// verifyOwnership verifica que el usuario sea propietario del archivo (para modo no recursivo)
func verifyOwnership(file *os.File, superblock *structs.Superblock, inodeNum int32, session *structs.Session, partition mounted.MountedPartition) error {
	// Root puede cambiar cualquier archivo
	if session.User == "root" {
		return nil
	}

	// Leer el inodo del archivo/carpeta
	inode, err := reports.ReadInodeAt(file, inodeNum, superblock.S_inode_start)
	if err != nil {
		return fmt.Errorf("error leyendo inodo: %v", err)
	}

	// Obtener información del usuario actual
	currentUser, err := users.GetCurrentUserInfo(session, partition)
	if err != nil {
		return fmt.Errorf("error obteniendo información del usuario: %v", err)
	}

	// Verificar propiedad
	if inode.I_uid != currentUser.UID {
		return errors.New("no tiene permisos para cambiar los permisos de este archivo (no es el propietario)")
	}

	return nil
}

// changePermissionsRecursive cambia los permisos del archivo/carpeta especificado
func changePermissionsRecursive(file *os.File, superblock *structs.Superblock, inodeNum int32, permissions int32, recursive bool, session *structs.Session, partition mounted.MountedPartition) error {
	// Leer el inodo
	inode, err := reports.ReadInodeAt(file, inodeNum, superblock.S_inode_start)
	if err != nil {
		return fmt.Errorf("error leyendo inodo: %v", err)
	}

	// Si no es root y es modo recursivo, verificar propiedad antes de cambiar
	if recursive && session.User != "root" {
		// Obtener información del usuario actual
		currentUser, err := users.GetCurrentUserInfo(session, partition)
		if err != nil {
			return fmt.Errorf("error obteniendo información del usuario: %v", err)
		}

		// Solo cambiar si es propietario
		if inode.I_uid != currentUser.UID {
			// En modo recursivo, simplemente saltar archivos que no son propios
			return nil
		}
	}

	// Cambiar los permisos del inodo actual
	inode.I_perm = permissions

	// Escribir el inodo actualizado al disco
	inodePos := int64(superblock.S_inode_start + (inodeNum * int32(binary.Size(structs.Inode{}))))
	_, err = file.Seek(inodePos, 0)
	if err != nil {
		return fmt.Errorf("error posicionando archivo para escribir inodo: %v", err)
	}

	err = binary.Write(file, binary.LittleEndian, inode)
	if err != nil {
		return fmt.Errorf("error escribiendo inodo actualizado: %v", err)
	}

	// Si es una carpeta y se especificó recursivo, cambiar también el contenido
	if inode.I_type == 1 && recursive {
		err = changePermissionsInDirectory(file, superblock, inode, permissions, session, partition)
		if err != nil {
			return fmt.Errorf("error cambiando permisos recursivamente: %v", err)
		}
	}

	return nil
}

// changePermissionsInDirectory cambia los permisos de todos los archivos en una carpeta recursivamente
func changePermissionsInDirectory(file *os.File, superblock *structs.Superblock, dirInode *structs.Inode, permissions int32, session *structs.Session, partition mounted.MountedPartition) error {
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

			// Cambiar permisos recursivamente
			err = changePermissionsRecursive(file, superblock, entry.B_inodo, permissions, true, session, partition)
			if err != nil {
				// Continuar con otros archivos aunque uno falle
				continue
			}
		}
	}

	return nil
}
