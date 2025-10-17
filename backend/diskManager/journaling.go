package diskmanager

import (
	"fmt"
	"main/utils/journaling"
	"main/utils/recovery"
	"time"
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

	// Paso 1: Leer las entradas del journal desde el disco
	fmt.Println("Paso 1: Leyendo entradas del journal desde el disco...")
	journalEntries, err := recovery.ReadJournalFromDisk(diskPath, partitionName)
	if err != nil {
		return fmt.Errorf("error leyendo journal desde disco: %v", err)
	}

	fmt.Printf("Se encontraron %d entradas en el journal\n", len(journalEntries))

	// Paso 2: Validar la integridad del journal
	fmt.Println("Paso 2: Validando integridad del journal...")
	if !recovery.ValidateJournalIntegrity(journalEntries) {
		fmt.Println("ADVERTENCIA: Se detectaron inconsistencias en el journal")
	} else {
		fmt.Println("Journal validado correctamente")
	}

	// Paso 3: Aplicar rollback de transacciones incompletas
	fmt.Println("Paso 3: Aplicando rollback de transacciones incompletas...")
	err = recovery.RollbackIncompleteTransactions(diskPath, partitionName, journalEntries)
	if err != nil {
		return fmt.Errorf("error durante rollback de transacciones: %v", err)
	}

	// Paso 4: Restaurar la consistencia del sistema de archivos
	fmt.Println("Paso 4: Restaurando consistencia del sistema de archivos...")
	err = recovery.RestoreFileSystemConsistency(diskPath, partitionName)
	if err != nil {
		return fmt.Errorf("error restaurando consistencia: %v", err)
	}

	// Paso 5: Registrar la recuperación en el journal en memoria
	journaling.AddJournalEntryForPartition(partitionName, "RECOVERY", fmt.Sprintf("Partición %s", partitionName), "Recuperación del sistema de archivos EXT3 completada exitosamente")

	fmt.Printf("Sistema de archivos EXT3 recuperado exitosamente para partición %s\n", partitionName)
	fmt.Printf("Resumen de la recuperación:\n")
	fmt.Printf("- Entradas del journal procesadas: %d\n", len(journalEntries))
	fmt.Printf("- Estado del sistema: Consistente\n")
	fmt.Printf("- Fecha de recuperación: %s\n", time.Now().Format("2006-01-02 15:04:05"))

	return nil
}

// InitializeJournalForEXT3 inicializa el journal cuando se formatea una partición como EXT3
func InitializeJournalForEXT3(partitionID string) {
	journaling.AddJournalEntryForPartition(partitionID, "FORMAT_EXT3", "/", fmt.Sprintf("Partición %s formateada como EXT3 - Journal inicializado", partitionID))
	fmt.Printf("Journal inicializado para partición EXT3: %s\n", partitionID)
}
