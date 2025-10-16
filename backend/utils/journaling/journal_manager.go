package journaling

import (
	"fmt"
	"sync"
	"time"
)

type JournalEntry struct {
	PartitionID string    `json:"partition_id"`
	Operation   string    `json:"operation"`
	Path        string    `json:"path"`
	Content     string    `json:"content"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"` // "PENDING", "COMMITTED", "ABORTED"
}

type JournalingInfo struct {
	PartitionID string         `json:"partition_id"`
	Entries     []JournalEntry `json:"entries"`
	TotalCount  int            `json:"total_count"`
}

// Journal manager para manejar entradas por partición
type JournalManager struct {
	entries map[string][]JournalEntry // map[partitionID][]JournalEntry
	mutex   sync.RWMutex
}

var journalManager = &JournalManager{
	entries: make(map[string][]JournalEntry),
}

// LogEXT3Operation registra automáticamente operaciones del sistema de archivos EXT3
func LogEXT3Operation(partitionID, operation, filePath, details string) {
	// Solo registrar si es una partición EXT3
	if IsEXT3Partition(partitionID) {
		AddJournalEntryForPartition(partitionID, operation, filePath, details)
	}
}

// AddJournalEntryForPartition agrega una entrada al journaling para una partición específica
func AddJournalEntryForPartition(partitionID, operation, path, content string) {
	journalManager.mutex.Lock()
	defer journalManager.mutex.Unlock()

	entry := JournalEntry{
		PartitionID: partitionID,
		Operation:   operation,
		Path:        path,
		Content:     content,
		Timestamp:   time.Now(),
		Status:      "COMMITTED",
	}

	if journalManager.entries[partitionID] == nil {
		journalManager.entries[partitionID] = make([]JournalEntry, 0)
	}

	journalManager.entries[partitionID] = append(journalManager.entries[partitionID], entry)

	// Log para debugging
	fmt.Printf("JOURNAL [%s]: %s en %s - %s\n", partitionID, operation, path, entry.Timestamp.Format("15:04:05"))
}

// IsEXT3Partition verifica si una partición es EXT3 (simplificado por ahora)
func IsEXT3Partition(partitionID string) bool {
	// Por ahora asumir que todas las particiones son EXT3
	// En una implementación real, verificarías el tipo del sistema de archivos
	return true
}

// GetJournalingInfo obtiene la información del journaling para una partición específica
func GetJournalingInfo(partitionID string) JournalingInfo {
	journalManager.mutex.RLock()
	defer journalManager.mutex.RUnlock()

	entries := journalManager.entries[partitionID]
	if entries == nil {
		entries = []JournalEntry{}
	}

	return JournalingInfo{
		PartitionID: partitionID,
		Entries:     entries,
		TotalCount:  len(entries),
	}
}

// ShowJournal muestra todas las entradas del journaling.
func ShowJournal() {
	journalManager.mutex.RLock()
	defer journalManager.mutex.RUnlock()

	fmt.Println("=== JOURNAL COMPLETO ===")
	for partitionID, entries := range journalManager.entries {
		fmt.Printf("\n--- Partición: %s ---\n", partitionID)
		for i, entry := range entries {
			fmt.Printf("[%d] %s | %s | %s | %s | %s\n",
				i+1,
				entry.Timestamp.Format("2006-01-02 15:04:05"),
				entry.Operation,
				entry.Path,
				entry.Content,
				entry.Status)
		}
	}
}

// ClearJournal limpia todas las entradas del journal para todas las particiones
func ClearJournal() {
	journalManager.mutex.Lock()
	defer journalManager.mutex.Unlock()
	journalManager.entries = make(map[string][]JournalEntry)
}

// ClearJournalForPartition limpia las entradas del journal para una partición específica
func ClearJournalForPartition(partitionID string) {
	journalManager.mutex.Lock()
	defer journalManager.mutex.Unlock()
	delete(journalManager.entries, partitionID)
}
