package mounted

import (
	"fmt"
)

func AddMountedPartition(mm *MountManager, path, name string, carnet string) (string, int, error) {
	mm.Mutex.Lock()
	defer mm.Mutex.Unlock()

	suffix := carnet[len(carnet)-2:]

	var letter byte
	var counter int

	diskLetter, diskExists := mm.DiskLetters[path]

	if diskExists {

		letter = diskLetter
		mm.DiskCounters[path]++
		counter = mm.DiskCounters[path]
	} else {

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

		mm.DiskCounters[path] = 1
		counter = 1
	}

	id := suffix + fmt.Sprintf("%d", counter) + string(letter)

	mountedPartition := MountedPartition{
		Path:        path,
		Name:        name,
		ID:          id,
		Correlative: counter,
		Letter:      letter,
	}

	mm.MountedPartitions = append(mm.MountedPartitions, mountedPartition)

	return id, counter, nil
}
