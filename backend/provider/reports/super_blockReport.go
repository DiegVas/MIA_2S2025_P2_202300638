package reports

import (
	"fmt"
	"main/provider/common"
	"main/provider/mounted"

	"strings"
)

func generateSuperblockReport(partitionID string) (string, error) {
	// Obtener información de la partición montada
	mountManager := mounted.GetMountManager()
	mountedPartition, found := mountManager.GetMountedPartition(partitionID)
	if !found {
		return "", fmt.Errorf("partición con ID '%s' no está montada", partitionID)
	}

	// Leer el superbloque para obtener información del sistema de archivos
	superblock, err := common.ReadSuperblock(mountedPartition.Path, partitionID)
	if err != nil {
		return "", fmt.Errorf("error leyendo superbloque: %v", err)
	}

	// Generar contenido DOT para el reporte del superbloque
	var dot strings.Builder

	// Iniciar el grafo
	dot.WriteString("digraph Superblock_Report {\n")
	dot.WriteString("    rankdir=TB;\n")
	dot.WriteString("    node [shape=record, fontname=\"Arial\"];\n")
	dot.WriteString("    edge [fontname=\"Arial\"];\n")
	dot.WriteString("    graph [fontname=\"Arial\"];\n\n")

	// Título del reporte
	dot.WriteString(fmt.Sprintf("    title [label=\"SuperBloque Particion %s\\nen disco %s\", shape=box, style=filled, fillcolor=lightblue];\n\n",
		partitionID, mountedPartition.Path))

	// Crear la tabla del superbloque
	dot.WriteString("    superblock [label=\"")
	dot.WriteString("{SUPERBLOQUE|")

	// Información del sistema de archivos
	fsType := "Desconocido"
	switch superblock.S_filesystem_type {
	case 2:
		fsType = "EXT2"
	case 3:
		fsType = "EXT3"
	}

	dot.WriteString(fmt.Sprintf("Tipo de Sistema de Archivos|%s\\l|", fsType))
	dot.WriteString(fmt.Sprintf("Número total de inodos|%d\\l|", superblock.S_inodes_count))
	dot.WriteString(fmt.Sprintf("Número total de bloques|%d\\l|", superblock.S_blocks_count))
	dot.WriteString(fmt.Sprintf("Número de bloques libres|%d\\l|", superblock.S_free_blocks_count))
	dot.WriteString(fmt.Sprintf("Número de inodos libres|%d\\l|", superblock.S_free_inodes_count))

	// Fechas de montaje (convertir de bytes a string)
	mtimeStr := strings.TrimRight(string(superblock.S_mtime[:]), "\x00")
	if mtimeStr == "" {
		mtimeStr = "No definido"
	}
	umtimeStr := strings.TrimRight(string(superblock.S_umtime[:]), "\x00")
	if umtimeStr == "" {
		umtimeStr = "No definido"
	}

	dot.WriteString(fmt.Sprintf("Última fecha de montaje|%s\\l|", mtimeStr))

	// Número mágico
	magicValid := "Inválido"
	if superblock.S_magic == 0xEF53 {
		magicValid = "Válido (0xEF53)"
	} else {
		magicValid = fmt.Sprintf("Inválido (0x%X)", superblock.S_magic)
	}
	dot.WriteString(fmt.Sprintf("Número mágico|%s\\l|", magicValid))

	// Tamanos
	dot.WriteString(fmt.Sprintf("Tamano del inodo|%d bytes\\l|", superblock.S_inode_size))
	dot.WriteString(fmt.Sprintf("Tamano del bloque|%d bytes\\l|", superblock.S_block_size))

	// Posiciones de inicio
	dot.WriteString(fmt.Sprintf("Primer inodo libre|%d\\l|", superblock.S_first_ino))
	dot.WriteString(fmt.Sprintf("Primer bloque libre|%d\\l|", superblock.S_first_blo))
	dot.WriteString(fmt.Sprintf("Inicio bitmap de inodos|%d\\l|", superblock.S_bm_inode_start))
	dot.WriteString(fmt.Sprintf("Inicio bitmap de bloques|%d\\l|", superblock.S_bm_block_start))
	dot.WriteString(fmt.Sprintf("Inicio tabla de inodos|%d\\l|", superblock.S_inode_start))
	dot.WriteString(fmt.Sprintf("Inicio tabla de bloques|%d\\l", superblock.S_block_start))

	dot.WriteString("}\", style=filled, fillcolor=lightgray];\n\n")

	// Conectar título con la tabla
	dot.WriteString("    title -> superblock;\n\n")

	// Cerrar el grafo
	dot.WriteString("}\n")

	return dot.String(), nil
}
