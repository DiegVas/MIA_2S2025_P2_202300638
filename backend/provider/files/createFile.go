package files

import (
	"errors"
	"fmt"
	"os"
	"strings"

	diskio "main/diskManager"
	"main/provider/common"
	"main/provider/mounted"
	"main/structs"
	"main/utils/journaling"
)

func CreateFile(params *structs.MkfileParams) error {

	session := structs.GetSession()
	if !session.Active {
		return errors.New("debe iniciar sesión para usar el comando MKFILE")
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

	// ! Buscar la partición montada
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

	// ? Abrir el archivo del disco para escritura
	file, err := os.OpenFile(partition.Path, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("error al abrir el disco: %v", err)
	}
	defer file.Close()

	// Leer el superbloque
	superblock, err := common.ReadSuperblock(partition.Path, session.ID)
	if err != nil {
		return fmt.Errorf("error leyendo superbloque: %v", err)
	}

	// Detectar el tipo de sistema de archivos y obtener posiciones
	startPosition := targetPartition.Part_start
	positions, err := common.GetFilesystemPositions(file, startPosition)
	if err != nil {
		return fmt.Errorf("error al obtener posiciones del sistema de archivos: %v", err)
	}

	// Usar las posiciones del superbloque
	bmInodePos := positions.BmInodePos
	bmBlockPos := positions.BmBlockPos
	inodePos := positions.InodePos
	blockPos := positions.BlockPos
	inodeCount := positions.InodeCount
	blockCount := positions.BlockCount

	fmt.Printf("DEBUG CreateFile: filesystem_type=%d, usando posiciones del superbloque\n", positions.FilesystemType)
	fmt.Printf("DEBUG CreateFile: inodePos=%d, blockPos=%d\n", inodePos, blockPos)

	// ? Preparar el contenido del archivo
	content, contentSize, err := prepareFileContent(params)
	if err != nil {
		return err
	}

	// ? Verificar y crear el archivo
	err = createFileInFilesystem(file, params.Path, content, contentSize, session, superblock,
		inodePos, blockPos, bmInodePos, bmBlockPos, inodeCount, blockCount, params.R)
	if err != nil {
		return err
	}

	// Registrar la operación en el journal EXT3
	journaling.LogEXT3Operation(session.ID, "CREATE_FILE", params.Path,
		fmt.Sprintf("Archivo creado con tamaño %d bytes", contentSize))

	return nil
}
