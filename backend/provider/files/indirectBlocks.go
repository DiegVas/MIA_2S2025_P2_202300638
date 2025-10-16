package files

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"main/structs"
	"os"
	"strings"
)

// AddFileToDirectoryWithIndirect maneja bloques indirectos cuando los directos están llenos
func AddFileToDirectoryWithIndirect(file *os.File, fileName string, fileInodeNum, dirInodeNum int32, superblock *structs.Superblock,
	inodePos, blockPos, bmBlockPos, blockCount int32) error {

	fmt.Printf("DEBUG AddFileToDirectoryWithIndirect: Añadiendo '%s' (inodo %d) al directorio %d\n", fileName, fileInodeNum, dirInodeNum)

	// Leer el inodo del directorio usando superblock.S_inode_start
	inodeBytes := make([]byte, structs.INODE_SIZE)
	inodeOffset := int64(superblock.S_inode_start + (dirInodeNum * structs.INODE_SIZE))

	_, err := file.ReadAt(inodeBytes, inodeOffset)
	if err != nil {
		return err
	}

	var dirInode structs.Inode
	buffer := bytes.NewReader(inodeBytes)
	binary.Read(buffer, binary.LittleEndian, &dirInode)

	// CASO ESPECIAL: El directorio raíz (inodo 0) usa el bloque 0 aunque aparezca como 0 en I_block
	if dirInodeNum == 0 && dirInode.I_block[0] == 0 {
		// Tratamos el bloque 0 como si ya estuviera asignado al directorio raíz
	}

	// 1. FASE 1: Buscar en bloques directos (0-11)
	fmt.Printf("DEBUG AddFileToDirectoryWithIndirect: Fase 1 - Buscando en bloques directos\n")
	found, err := addToDirectBlocks(file, fileName, fileInodeNum, &dirInode, dirInodeNum, superblock, bmBlockPos, blockCount)
	if err != nil {
		fmt.Printf("DEBUG AddFileToDirectoryWithIndirect: Error en bloques directos: %v\n", err)
		return err
	}
	if found {
		fmt.Printf("DEBUG AddFileToDirectoryWithIndirect: ¡Añadido en bloques directos!\n")
		// Actualizar el inodo usando superblock.S_inode_start
		return updateDirectoryInode(file, &dirInode, dirInodeNum, superblock.S_inode_start)
	}
	fmt.Printf("DEBUG AddFileToDirectoryWithIndirect: No se pudo añadir en bloques directos\n")

	// 2. FASE 2: Buscar en bloque indirecto simple (slot 12)
	found, err = addToIndirectBlock(file, fileName, fileInodeNum, &dirInode, 12, blockPos, bmBlockPos, blockCount)
	if err != nil {
		return err
	}
	if found {
		return updateDirectoryInode(file, &dirInode, dirInodeNum, superblock.S_inode_start)
	}

	// 3. FASE 3: Buscar en bloque doblemente indirecto (slot 13)
	found, err = addToDoubleIndirectBlock(file, fileName, fileInodeNum, &dirInode, 13, blockPos, bmBlockPos, blockCount)
	if err != nil {
		return err
	}
	if found {
		return updateDirectoryInode(file, &dirInode, dirInodeNum, superblock.S_inode_start)
	}

	// 4. FASE 4: Buscar en bloque triplemente indirecto (slot 14)
	found, err = addToTripleIndirectBlock(file, fileName, fileInodeNum, &dirInode, 14, blockPos, bmBlockPos, blockCount)
	if err != nil {
		return err
	}
	if found {
		return updateDirectoryInode(file, &dirInode, dirInodeNum, superblock.S_inode_start)
	}

	return errors.New("directorio completamente lleno - no se puede agregar más entradas")
}

// addToDirectBlocks busca espacio en los primeros 12 bloques directos
func addToDirectBlocks(file *os.File, fileName string, fileInodeNum int32,
	dirInode *structs.Inode, dirInodeNum int32, superblock *structs.Superblock, bmBlockPos, blockCount int32) (bool, error) {

	fmt.Printf("DEBUG addToDirectBlocks: Revisando bloques directos para directorio %d\n", dirInodeNum)
	for i := 0; i < 12; i++ {
		fmt.Printf("DEBUG addToDirectBlocks: Slot %d, bloque=%d\n", i, dirInode.I_block[i])

		// CASO ESPECIAL: Para el directorio raíz, el slot 0 con bloque 0 es válido
		isRootSpecialCase := (dirInodeNum == 0 && i == 0 && dirInode.I_block[i] == 0)

		if dirInode.I_block[i] != 0 || isRootSpecialCase {
			blockNum := dirInode.I_block[i]
			fmt.Printf("DEBUG addToDirectBlocks: Slot %d ocupado, bloque=%d\n", i, blockNum)

			// Verificar si hay espacio en este bloque existente
			var blockPos int32
			if superblock.S_block_start == 0 {
				blockPos = superblock.S_inode_start + (superblock.S_inodes_count * structs.INODE_SIZE)
			} else {
				blockPos = superblock.S_block_start
			}
			found, err := addToExistingBlock(file, fileName, fileInodeNum, blockNum, blockPos)
			if err != nil {
				fmt.Printf("DEBUG addToDirectBlocks: Error en bloque existente %d: %v\n", blockNum, err)
				continue
			}
			if found {
				fmt.Printf("DEBUG addToDirectBlocks: ¡Añadido a bloque existente %d!\n", blockNum)
				return true, nil
			}
			fmt.Printf("DEBUG addToDirectBlocks: Bloque %d está lleno\n", blockNum)
		} else {
			fmt.Printf("DEBUG addToDirectBlocks: Slot %d libre, creando nuevo bloque\n", i)
			// Slot libre - crear nuevo bloque
			var blockPos int32
			if superblock.S_block_start == 0 {
				blockPos = superblock.S_inode_start + (superblock.S_inodes_count * structs.INODE_SIZE)
			} else {
				blockPos = superblock.S_block_start
			}
			newBlockNum, err := createNewFolderBlockWithData(file, fileName, fileInodeNum, blockPos, bmBlockPos, blockCount)
			if err != nil {
				fmt.Printf("DEBUG addToDirectBlocks: Error creando nuevo bloque: %v\n", err)
				return false, err
			}
			fmt.Printf("DEBUG addToDirectBlocks: ¡Nuevo bloque creado! Bloque=%d\n", newBlockNum)
			dirInode.I_block[i] = newBlockNum
			return true, nil
		}
	}
	return false, nil
}

// addToIndirectBlock maneja bloques indirectos simples
func addToIndirectBlock(file *os.File, fileName string, fileInodeNum int32,
	dirInode *structs.Inode, slotIndex int, blockPos, bmBlockPos, blockCount int32) (bool, error) {

	if dirInode.I_block[slotIndex] == 0 {
		// Crear nuevo bloque de punteros
		newBlocks, err := FindFreeBlocks(file, bmBlockPos, blockCount, 1)
		if err != nil {
			return false, err
		}
		dirInode.I_block[slotIndex] = newBlocks[0]

		// Inicializar bloque de punteros vacío
		var pointerBlock structs.PointerBlock
		var buffer bytes.Buffer
		binary.Write(&buffer, binary.LittleEndian, pointerBlock)
		_, err = file.WriteAt(buffer.Bytes(), int64(blockPos+(newBlocks[0]*structs.BLOCK_SIZE)))
		if err != nil {
			return false, err
		}

		err = MarkBlockAsUsed(file, bmBlockPos, newBlocks[0])
		if err != nil {
			return false, err
		}
	}

	// Leer bloque de punteros
	blockBytes := make([]byte, structs.BLOCK_SIZE)
	_, err := file.ReadAt(blockBytes, int64(blockPos+(dirInode.I_block[slotIndex]*structs.BLOCK_SIZE)))
	if err != nil {
		return false, err
	}

	var pointerBlock structs.PointerBlock
	buffer := bytes.NewReader(blockBytes)
	binary.Read(buffer, binary.LittleEndian, &pointerBlock)

	// Buscar en bloques apuntados
	for i := 0; i < 16; i++ {
		if pointerBlock.B_pointers[i] != 0 {
			found, err := addToExistingBlock(file, fileName, fileInodeNum, pointerBlock.B_pointers[i], blockPos)
			if err != nil {
				continue
			}
			if found {
				return true, nil
			}
		} else {
			// Crear nuevo bloque de carpeta
			newBlockNum, err := createNewFolderBlockWithData(file, fileName, fileInodeNum, blockPos, bmBlockPos, blockCount)
			if err != nil {
				return false, err
			}
			pointerBlock.B_pointers[i] = newBlockNum

			// Actualizar bloque de punteros
			var ptrBuffer bytes.Buffer
			binary.Write(&ptrBuffer, binary.LittleEndian, pointerBlock)
			_, err = file.WriteAt(ptrBuffer.Bytes(), int64(blockPos+(dirInode.I_block[slotIndex]*structs.BLOCK_SIZE)))
			return true, err
		}
	}
	return false, nil
}

// addToDoubleIndirectBlock maneja bloques doblemente indirectos
func addToDoubleIndirectBlock(file *os.File, fileName string, fileInodeNum int32,
	dirInode *structs.Inode, slotIndex int, blockPos, bmBlockPos, blockCount int32) (bool, error) {

	if dirInode.I_block[slotIndex] == 0 {
		// Crear nuevo bloque de punteros de primer nivel
		newBlocks, err := FindFreeBlocks(file, bmBlockPos, blockCount, 1)
		if err != nil {
			return false, err
		}
		dirInode.I_block[slotIndex] = newBlocks[0]

		// Inicializar bloque de punteros vacío
		var pointerBlock structs.PointerBlock
		var buffer bytes.Buffer
		binary.Write(&buffer, binary.LittleEndian, pointerBlock)
		_, err = file.WriteAt(buffer.Bytes(), int64(blockPos+(newBlocks[0]*structs.BLOCK_SIZE)))
		if err != nil {
			return false, err
		}

		err = MarkBlockAsUsed(file, bmBlockPos, newBlocks[0])
		if err != nil {
			return false, err
		}
	}

	// Leer bloque de punteros de primer nivel
	blockBytes := make([]byte, structs.BLOCK_SIZE)
	_, err := file.ReadAt(blockBytes, int64(blockPos+(dirInode.I_block[slotIndex]*structs.BLOCK_SIZE)))
	if err != nil {
		return false, err
	}

	var firstLevelPointers structs.PointerBlock
	buffer := bytes.NewReader(blockBytes)
	binary.Read(buffer, binary.LittleEndian, &firstLevelPointers)

	// Recorrer punteros de primer nivel (cada uno apunta a un bloque de punteros de segundo nivel)
	for i := 0; i < 16; i++ {
		if firstLevelPointers.B_pointers[i] != 0 {
			// Buscar en el bloque de segundo nivel existente
			found, err := searchInSecondLevelBlock(file, fileName, fileInodeNum,
				firstLevelPointers.B_pointers[i], blockPos, bmBlockPos, blockCount)
			if err != nil {
				continue
			}
			if found {
				return true, nil
			}
		} else {
			// Crear nuevo bloque de punteros de segundo nivel
			newBlocks, err := FindFreeBlocks(file, bmBlockPos, blockCount, 1)
			if err != nil {
				continue
			}

			secondLevelBlockNum := newBlocks[0]
			firstLevelPointers.B_pointers[i] = secondLevelBlockNum

			// Inicializar bloque de segundo nivel
			var secondLevelPointers structs.PointerBlock
			var secBuffer bytes.Buffer
			binary.Write(&secBuffer, binary.LittleEndian, secondLevelPointers)
			_, err = file.WriteAt(secBuffer.Bytes(), int64(blockPos+(secondLevelBlockNum*structs.BLOCK_SIZE)))
			if err != nil {
				return false, err
			}

			err = MarkBlockAsUsed(file, bmBlockPos, secondLevelBlockNum)
			if err != nil {
				return false, err
			}

			// Crear primer bloque de carpeta en este nuevo segundo nivel
			newFolderBlockNum, err := createNewFolderBlockWithData(file, fileName, fileInodeNum, blockPos, bmBlockPos, blockCount)
			if err != nil {
				return false, err
			}

			// Actualizar el primer puntero del bloque de segundo nivel
			secondLevelPointers.B_pointers[0] = newFolderBlockNum
			var secUpdateBuffer bytes.Buffer
			binary.Write(&secUpdateBuffer, binary.LittleEndian, secondLevelPointers)
			_, err = file.WriteAt(secUpdateBuffer.Bytes(), int64(blockPos+(secondLevelBlockNum*structs.BLOCK_SIZE)))
			if err != nil {
				return false, err
			}

			// Actualizar el bloque de primer nivel
			var firstUpdateBuffer bytes.Buffer
			binary.Write(&firstUpdateBuffer, binary.LittleEndian, firstLevelPointers)
			_, err = file.WriteAt(firstUpdateBuffer.Bytes(), int64(blockPos+(dirInode.I_block[slotIndex]*structs.BLOCK_SIZE)))
			return true, err
		}
	}

	return false, nil
}

// searchInSecondLevelBlock busca espacio en un bloque de punteros de segundo nivel
func searchInSecondLevelBlock(file *os.File, fileName string, fileInodeNum, secondLevelBlockNum,
	blockPos, bmBlockPos, blockCount int32) (bool, error) {

	// Leer bloque de punteros de segundo nivel
	blockBytes := make([]byte, structs.BLOCK_SIZE)
	_, err := file.ReadAt(blockBytes, int64(blockPos+(secondLevelBlockNum*structs.BLOCK_SIZE)))
	if err != nil {
		return false, err
	}

	var secondLevelPointers structs.PointerBlock
	buffer := bytes.NewReader(blockBytes)
	binary.Read(buffer, binary.LittleEndian, &secondLevelPointers)

	// Buscar en bloques de carpeta apuntados
	for i := 0; i < 16; i++ {
		if secondLevelPointers.B_pointers[i] != 0 {
			found, err := addToExistingBlock(file, fileName, fileInodeNum, secondLevelPointers.B_pointers[i], blockPos)
			if err != nil {
				continue
			}
			if found {
				return true, nil
			}
		} else {
			// Crear nuevo bloque de carpeta
			newBlockNum, err := createNewFolderBlockWithData(file, fileName, fileInodeNum, blockPos, bmBlockPos, blockCount)
			if err != nil {
				return false, err
			}

			secondLevelPointers.B_pointers[i] = newBlockNum

			// Actualizar bloque de punteros de segundo nivel
			var ptrBuffer bytes.Buffer
			binary.Write(&ptrBuffer, binary.LittleEndian, secondLevelPointers)
			_, err = file.WriteAt(ptrBuffer.Bytes(), int64(blockPos+(secondLevelBlockNum*structs.BLOCK_SIZE)))
			return true, err
		}
	}

	return false, nil
}

// Funciones auxiliares
func addToExistingBlock(file *os.File, fileName string, fileInodeNum, blockNum, blockPos int32) (bool, error) {
	fmt.Printf("DEBUG addToExistingBlock: Intentando añadir '%s' (inodo %d) al bloque %d\n", fileName, fileInodeNum, blockNum)

	blockBytes := make([]byte, structs.BLOCK_SIZE)

	// TEMPORAL: usar blockPos tal como viene por ahora
	// TODO: Esta función debería recibir el superblock para calcular correctamente
	blockPosition := int64(blockPos + (blockNum * structs.BLOCK_SIZE))

	fmt.Printf("DEBUG addToExistingBlock: Leyendo bloque %d en posición %d\n", blockNum, blockPosition)
	_, err := file.ReadAt(blockBytes, blockPosition)
	if err != nil {
		fmt.Printf("DEBUG addToExistingBlock: Error leyendo bloque: %v\n", err)
		return false, err
	}

	var folderBlock structs.FolderBlock
	buffer := bytes.NewReader(blockBytes)
	binary.Read(buffer, binary.LittleEndian, &folderBlock)

	fmt.Printf("DEBUG addToExistingBlock: Revisando entradas en bloque %d\n", blockNum)
	for j := range folderBlock.B_content {
		entryName := strings.TrimRight(string(folderBlock.B_content[j].B_name[:]), "\x00")
		fmt.Printf("DEBUG addToExistingBlock: Entrada %d: nombre='%s', inodo=%d\n", j, entryName, folderBlock.B_content[j].B_inodo)

		if entryName == "" || folderBlock.B_content[j].B_inodo == 0 {
			fmt.Printf("DEBUG addToExistingBlock: Encontrado slot libre en posición %d\n", j)
			copy(folderBlock.B_content[j].B_name[:], fileName)
			folderBlock.B_content[j].B_inodo = fileInodeNum

			var folderBuffer bytes.Buffer
			binary.Write(&folderBuffer, binary.LittleEndian, folderBlock)
			_, err = file.WriteAt(folderBuffer.Bytes(), int64(blockPosition))
			if err != nil {
				fmt.Printf("DEBUG addToExistingBlock: Error escribiendo bloque: %v\n", err)
				return false, err
			}
			fmt.Printf("DEBUG addToExistingBlock: ¡Éxito! Añadido '%s' en slot %d del bloque %d\n", fileName, j, blockNum)
			return true, nil
		}
	}

	fmt.Printf("DEBUG addToExistingBlock: Bloque %d está lleno\n", blockNum)
	return false, nil
}

// createNewFolderBlockWithData crea un nuevo bloque de carpeta con los datos proporcionados
func createNewFolderBlockWithData(file *os.File, fileName string, fileInodeNum, blockPos, bmBlockPos, blockCount int32) (int32, error) {
	// Encontrar un bloque libre
	freeBlocks, err := FindFreeBlocks(file, bmBlockPos, blockCount, 1)
	if err != nil {
		return 0, err
	}

	newBlockNum := freeBlocks[0]

	// Crear nuevo FolderBlock con la entrada
	var newFolderBlock structs.FolderBlock
	copy(newFolderBlock.B_content[0].B_name[:], fileName)
	newFolderBlock.B_content[0].B_inodo = fileInodeNum

	// Escribir el nuevo bloque
	var folderBuffer bytes.Buffer
	binary.Write(&folderBuffer, binary.LittleEndian, newFolderBlock)
	_, err = file.WriteAt(folderBuffer.Bytes(), int64(blockPos+(newBlockNum*structs.BLOCK_SIZE)))
	if err != nil {
		return 0, err
	}

	// Marcar el bloque como usado
	err = MarkBlockAsUsed(file, bmBlockPos, newBlockNum)
	if err != nil {
		return 0, err
	}

	return newBlockNum, nil
}

// addToTripleIndirectBlock maneja bloques triplemente indirectos
func addToTripleIndirectBlock(file *os.File, fileName string, fileInodeNum int32,
	dirInode *structs.Inode, slotIndex int, blockPos, bmBlockPos, blockCount int32) (bool, error) {

	if dirInode.I_block[slotIndex] == 0 {
		// Crear nuevo bloque de punteros de primer nivel
		newBlocks, err := FindFreeBlocks(file, bmBlockPos, blockCount, 1)
		if err != nil {
			return false, err
		}
		dirInode.I_block[slotIndex] = newBlocks[0]

		// Inicializar bloque de punteros vacío
		var pointerBlock structs.PointerBlock
		var buffer bytes.Buffer
		binary.Write(&buffer, binary.LittleEndian, pointerBlock)
		_, err = file.WriteAt(buffer.Bytes(), int64(blockPos+(newBlocks[0]*structs.BLOCK_SIZE)))
		if err != nil {
			return false, err
		}

		err = MarkBlockAsUsed(file, bmBlockPos, newBlocks[0])
		if err != nil {
			return false, err
		}
	}

	// Leer bloque de punteros de primer nivel
	blockBytes := make([]byte, structs.BLOCK_SIZE)
	_, err := file.ReadAt(blockBytes, int64(blockPos+(dirInode.I_block[slotIndex]*structs.BLOCK_SIZE)))
	if err != nil {
		return false, err
	}

	var firstLevelPointers structs.PointerBlock
	buffer := bytes.NewReader(blockBytes)
	binary.Read(buffer, binary.LittleEndian, &firstLevelPointers)

	// Recorrer punteros de primer nivel (cada uno apunta a un bloque de segundo nivel)
	for i := 0; i < 16; i++ {
		if firstLevelPointers.B_pointers[i] != 0 {
			// Buscar en el bloque de segundo nivel existente
			found, err := searchInTripleSecondLevel(file, fileName, fileInodeNum,
				firstLevelPointers.B_pointers[i], blockPos, bmBlockPos, blockCount)
			if err != nil {
				continue
			}
			if found {
				return true, nil
			}
		} else {
			// Crear nuevo bloque de punteros de segundo nivel
			newBlocks, err := FindFreeBlocks(file, bmBlockPos, blockCount, 1)
			if err != nil {
				continue
			}

			secondLevelBlockNum := newBlocks[0]
			firstLevelPointers.B_pointers[i] = secondLevelBlockNum

			// Inicializar bloque de segundo nivel
			var secondLevelPointers structs.PointerBlock
			var secBuffer bytes.Buffer
			binary.Write(&secBuffer, binary.LittleEndian, secondLevelPointers)
			_, err = file.WriteAt(secBuffer.Bytes(), int64(blockPos+(secondLevelBlockNum*structs.BLOCK_SIZE)))
			if err != nil {
				return false, err
			}

			err = MarkBlockAsUsed(file, bmBlockPos, secondLevelBlockNum)
			if err != nil {
				return false, err
			}

			// Crear primer bloque de tercer nivel
			found, err := initializeTripleThirdLevel(file, fileName, fileInodeNum,
				secondLevelBlockNum, blockPos, bmBlockPos, blockCount)
			if err != nil {
				return false, err
			}

			// Actualizar el bloque de primer nivel
			var firstUpdateBuffer bytes.Buffer
			binary.Write(&firstUpdateBuffer, binary.LittleEndian, firstLevelPointers)
			_, err = file.WriteAt(firstUpdateBuffer.Bytes(), int64(blockPos+(dirInode.I_block[slotIndex]*structs.BLOCK_SIZE)))
			return found, err
		}
	}

	return false, nil
}

// searchInTripleSecondLevel busca espacio en un bloque de segundo nivel para triple indirección
func searchInTripleSecondLevel(file *os.File, fileName string, fileInodeNum, secondLevelBlockNum,
	blockPos, bmBlockPos, blockCount int32) (bool, error) {

	// Leer bloque de punteros de segundo nivel
	blockBytes := make([]byte, structs.BLOCK_SIZE)
	_, err := file.ReadAt(blockBytes, int64(blockPos+(secondLevelBlockNum*structs.BLOCK_SIZE)))
	if err != nil {
		return false, err
	}

	var secondLevelPointers structs.PointerBlock
	buffer := bytes.NewReader(blockBytes)
	binary.Read(buffer, binary.LittleEndian, &secondLevelPointers)

	// Buscar en bloques de tercer nivel
	for i := 0; i < 16; i++ {
		if secondLevelPointers.B_pointers[i] != 0 {
			found, err := searchInSecondLevelBlock(file, fileName, fileInodeNum,
				secondLevelPointers.B_pointers[i], blockPos, bmBlockPos, blockCount)
			if err != nil {
				continue
			}
			if found {
				return true, nil
			}
		} else {
			// Crear nuevo bloque de tercer nivel
			found, err := initializeTripleThirdLevel(file, fileName, fileInodeNum,
				secondLevelBlockNum, blockPos, bmBlockPos, blockCount)
			if err != nil {
				return false, err
			}
			if found {
				return true, nil
			}
		}
	}

	return false, nil
}

// initializeTripleThirdLevel inicializa un nuevo bloque de tercer nivel
func initializeTripleThirdLevel(file *os.File, fileName string, fileInodeNum, secondLevelBlockNum,
	blockPos, bmBlockPos, blockCount int32) (bool, error) {

	// Crear nuevo bloque de punteros de tercer nivel
	newBlocks, err := FindFreeBlocks(file, bmBlockPos, blockCount, 1)
	if err != nil {
		return false, err
	}

	thirdLevelBlockNum := newBlocks[0]

	// Inicializar bloque de tercer nivel
	var thirdLevelPointers structs.PointerBlock
	var thirdBuffer bytes.Buffer
	binary.Write(&thirdBuffer, binary.LittleEndian, thirdLevelPointers)
	_, err = file.WriteAt(thirdBuffer.Bytes(), int64(blockPos+(thirdLevelBlockNum*structs.BLOCK_SIZE)))
	if err != nil {
		return false, err
	}

	err = MarkBlockAsUsed(file, bmBlockPos, thirdLevelBlockNum)
	if err != nil {
		return false, err
	}

	// Crear primer bloque de carpeta en este tercer nivel
	newFolderBlockNum, err := createNewFolderBlockWithData(file, fileName, fileInodeNum, blockPos, bmBlockPos, blockCount)
	if err != nil {
		return false, err
	}

	// Actualizar el primer puntero del bloque de tercer nivel
	thirdLevelPointers.B_pointers[0] = newFolderBlockNum
	var thirdUpdateBuffer bytes.Buffer
	binary.Write(&thirdUpdateBuffer, binary.LittleEndian, thirdLevelPointers)
	_, err = file.WriteAt(thirdUpdateBuffer.Bytes(), int64(blockPos+(thirdLevelBlockNum*structs.BLOCK_SIZE)))
	if err != nil {
		return false, err
	}

	// Actualizar el bloque de segundo nivel para apuntar al nuevo tercer nivel
	blockBytes := make([]byte, structs.BLOCK_SIZE)
	_, err = file.ReadAt(blockBytes, int64(blockPos+(secondLevelBlockNum*structs.BLOCK_SIZE)))
	if err != nil {
		return false, err
	}

	var secondLevelPointers structs.PointerBlock
	buffer := bytes.NewReader(blockBytes)
	binary.Read(buffer, binary.LittleEndian, &secondLevelPointers)

	// Encontrar el primer slot libre en el segundo nivel
	for i := 0; i < 16; i++ {
		if secondLevelPointers.B_pointers[i] == 0 {
			secondLevelPointers.B_pointers[i] = thirdLevelBlockNum
			break
		}
	}

	// Escribir el bloque de segundo nivel actualizado
	var secUpdateBuffer bytes.Buffer
	binary.Write(&secUpdateBuffer, binary.LittleEndian, secondLevelPointers)
	_, err = file.WriteAt(secUpdateBuffer.Bytes(), int64(blockPos+(secondLevelBlockNum*structs.BLOCK_SIZE)))

	return true, err
}

func updateDirectoryInode(file *os.File, dirInode *structs.Inode, dirInodeNum, inodePos int32) error {
	var inodeBuffer bytes.Buffer
	binary.Write(&inodeBuffer, binary.LittleEndian, dirInode)
	_, err := file.WriteAt(inodeBuffer.Bytes(), int64(inodePos+(dirInodeNum*structs.INODE_SIZE)))
	return err
}
