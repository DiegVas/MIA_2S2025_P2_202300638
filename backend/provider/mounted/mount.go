package mounted

import (
	"fmt"
	"sync"
)

type MountedPartition struct {
	Path        string // Ruta del disco
	DiskPath    string // Alias para Path (compatibilidad)
	Name        string
	ID          string
	Correlative int
	Letter      byte
}

type MountManager struct {
	MountedPartitions []MountedPartition
	DiskLetters       map[string]byte
	DiskCounters      map[string]int
	UsedCorrelatives  map[string]map[int]bool // Para rastrear correlativos usados por disco
	Mutex             sync.Mutex
}

func (mm *MountManager) AddMountedPartition(path, name string, carnet string) (string, int, error) {
	mm.Mutex.Lock()
	defer mm.Mutex.Unlock()

	suffix := carnet[len(carnet)-2:]

	var letter byte
	var correlative int

	diskLetter, diskExists := mm.DiskLetters[path]

	if diskExists {
		letter = diskLetter
		// Buscar el correlativo más bajo disponible
		correlative = mm.findLowestAvailableCorrelative(path)
	} else {
		// Asignar nueva letra para el disco
		lastLetter := byte('A' - 1)
		for _, l := range mm.DiskLetters {
			if l > lastLetter {
				lastLetter = l
			}
		}

		letter = lastLetter + 1
		if letter < 'A' {
			letter = 'A'
		}

		mm.DiskLetters[path] = letter
		mm.DiskCounters[path] = 0
		correlative = 1

		// Inicializar el mapa de correlativos para este disco
		if mm.UsedCorrelatives[path] == nil {
			mm.UsedCorrelatives[path] = make(map[int]bool)
		}
	}

	// Marcar el correlativo como usado
	if mm.UsedCorrelatives[path] == nil {
		mm.UsedCorrelatives[path] = make(map[int]bool)
	}
	mm.UsedCorrelatives[path][correlative] = true

	// Actualizar el contador máximo si es necesario
	if correlative > mm.DiskCounters[path] {
		mm.DiskCounters[path] = correlative
	}

	id := suffix + fmt.Sprintf("%d", correlative) + string(letter)

	mountedPartition := MountedPartition{
		Path:        path,
		DiskPath:    path, // Mantener compatibilidad
		Name:        name,
		ID:          id,
		Correlative: correlative,
		Letter:      letter,
	}

	mm.MountedPartitions = append(mm.MountedPartitions, mountedPartition)

	return id, correlative, nil
}

// findLowestAvailableCorrelative encuentra el correlativo más bajo disponible para un disco
func (mm *MountManager) findLowestAvailableCorrelative(diskPath string) int {
	if mm.UsedCorrelatives[diskPath] == nil {
		mm.UsedCorrelatives[diskPath] = make(map[int]bool)
		return 1
	}

	// Buscar el correlativo más bajo disponible
	for i := 1; i <= mm.DiskCounters[diskPath]+1; i++ {
		if !mm.UsedCorrelatives[diskPath][i] {
			return i
		}
	}

	// Si no se encontró ninguno disponible, retornar el siguiente
	return mm.DiskCounters[diskPath] + 1
}

func (mm *MountManager) GetMountedPartitions() []MountedPartition {
	mm.Mutex.Lock()
	defer mm.Mutex.Unlock()

	partitions := make([]MountedPartition, len(mm.MountedPartitions))
	copy(partitions, mm.MountedPartitions)

	return partitions
}

func (mm *MountManager) IsMounted(path, name string) bool {
	mm.Mutex.Lock()
	defer mm.Mutex.Unlock()

	for _, partition := range mm.MountedPartitions {
		if partition.Path == path && partition.Name == name {
			return true
		}
	}

	return false
}

func (mm *MountManager) GetIDMountedPartition(path, name string) (string, bool) {
	mm.Mutex.Lock()
	defer mm.Mutex.Unlock()

	for _, partition := range mm.MountedPartitions {
		if partition.Path == path && partition.Name == name {
			return partition.ID, true
		}
	}

	return "", false
}

func (mm *MountManager) GetMountedPartition(id string) (MountedPartition, bool) {
	mm.Mutex.Lock()
	defer mm.Mutex.Unlock()

	for _, partition := range mm.MountedPartitions {
		if partition.ID == id {
			return partition, true
		}
	}

	return MountedPartition{}, false
}

func (mm *MountManager) UnmountPartition(id string) bool {
	mm.Mutex.Lock()
	defer mm.Mutex.Unlock()

	for i, partition := range mm.MountedPartitions {
		if partition.ID == id {
			// Liberar el correlativo para reutilización
			if mm.UsedCorrelatives[partition.Path] != nil {
				mm.UsedCorrelatives[partition.Path][partition.Correlative] = false
			}

			// Eliminar la partición de la lista
			mm.MountedPartitions = append(mm.MountedPartitions[:i], mm.MountedPartitions[i+1:]...)
			return true
		}
	}

	return false
}
