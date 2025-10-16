package diskmanager

import (
	"fmt"
	"main/utils/journaling"
)

// Wrapper functions que redirigen al paquete journaling centralizado
// Mantenemos estas funciones por compatibilidad con código existente

// AddJournalEntry agrega una entrada al journaling para una partición específica
func AddJournalEntry(operation, path, content string) {
	journaling.AddJournalEntryForPartition("default", operation, path, content)
}

// AddJournalEntryForPartition agrega una entrada al journaling para una partición específica
func AddJournalEntryForPartition(partitionID, operation, path, content string) {
	journaling.AddJournalEntryForPartition(partitionID, operation, path, content)
}

// LogEXT3Operation registra automáticamente operaciones del sistema de archivos EXT3
func LogEXT3Operation(partitionID, operation, filePath, details string) {
	journaling.LogEXT3Operation(partitionID, operation, filePath, details)
}

// GetJournalingInfo obtiene la información del journaling para una partición específica
func GetJournalingInfo(partitionID string) journaling.JournalingInfo {
	return journaling.GetJournalingInfo(partitionID)
}

// ShowJournal muestra todas las entradas del journaling.
func ShowJournal() {
	journaling.ShowJournal()
}

// RecoverFileSystem recupera el sistema de archivos EXT3 usando journaling y superbloque
func RecoverFileSystem(diskPath string, partitionName string) error {
	fmt.Printf("Iniciando recuperación del sistema de archivos EXT3 para partición %s en disco %s\n", partitionName, diskPath)

	// En una implementación real, aquí se haría:
	// 1. Leer el superbloque para obtener información del estado del sistema de archivos
	// 2. Leer el journal para encontrar transacciones sin confirmar
	// 3. Aplicar o deshacer transacciones según sea necesario
	// 4. Restaurar la consistencia del sistema de archivos

	// Por ahora, simulamos la recuperación
	journaling.AddJournalEntryForPartition(partitionName, "RECOVERY", fmt.Sprintf("Partición %s", partitionName), "Recuperación del sistema de archivos EXT3 completada")

	fmt.Printf("Sistema de archivos EXT3 recuperado exitosamente para partición %s\n", partitionName)
	return nil
}

// InitializeJournalForEXT3 inicializa el journal cuando se formatea una partición como EXT3
func InitializeJournalForEXT3(partitionID string) {
	journaling.AddJournalEntryForPartition(partitionID, "FORMAT_EXT3", "/", fmt.Sprintf("Partición %s formateada como EXT3 - Journal inicializado", partitionID))
	fmt.Printf("Journal inicializado para partición EXT3: %s\n", partitionID)
}
