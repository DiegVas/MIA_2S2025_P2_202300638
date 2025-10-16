package structs

import "fmt"

type MkdiskParams struct {
	Size int
	Fit  string
	Unit byte
	Path string
}

func (p *MkdiskParams) GetSizeInBytes() int32 {
	switch p.Unit {
	case 'K':
		return int32(p.Size) * 1024
	case 'M':
		return int32(p.Size) * 1024 * 1024
	default:
		return int32(p.Size) * 1024 * 1024 // Por defecto Megabytes
	}
}

func (p *MkdiskParams) String() string {
	return fmt.Sprintf("Size: %d, Fit: %s, Unit: %c, Path: %s",
		p.Size, p.Fit, p.Unit, p.Path)
}

type FdiskParams struct {
	Size   int
	Unit   byte
	Path   string
	Type   byte
	Fit    string
	Name   string
	Delete string // "fast" o "full" para eliminar partición
	Add    int    // Cantidad a agregar/quitar (puede ser negativo)
}

func (p *FdiskParams) GetSizeInBytes() int32 {
	switch p.Unit {
	case 'B':
		return int32(p.Size)
	case 'K':
		return int32(p.Size) * 1024
	case 'M':
		return int32(p.Size) * 1024 * 1024
	default:
		return int32(p.Size) * 1024 // Por defecto Kilobytes
	}
}

func (p *FdiskParams) GetAddSizeInBytes() int32 {
	switch p.Unit {
	case 'B':
		return int32(p.Add)
	case 'K':
		return int32(p.Add) * 1024
	case 'M':
		return int32(p.Add) * 1024 * 1024
	default:
		return int32(p.Add) * 1024 // Por defecto Kilobytes
	}
}

type RepParams struct {
	Name       string // mbr, disk, inode, block, bm_inode, bm_block, tree, sb, file, ls
	Path       string // Ruta donde se guardará el reporte
	ID         string // ID de la partición montada
	PathFileLS string // Para reportes file y ls (opcional)
}

func (p *RepParams) String() string {
	return fmt.Sprintf("Name: %s, Path: %s, ID: %s, PathFileLS: %s",
		p.Name, p.Path, p.ID, p.PathFileLS)
}

func (p *FdiskParams) String() string {
	return fmt.Sprintf("Size: %d, Unit: %c, Path: %s, Type: %c, Fit: %s, Name: %s, Delete: %s, Add: %d",
		p.Size, p.Unit, p.Path, p.Type, p.Fit, p.Name, p.Delete, p.Add)
}

type MountParams struct {
	Path   string
	Name   string
	ID     string
	Carnet string // Carnet del estudiante, se usarán los últimos 2 dígitos
}

func (p *MountParams) String() string {
	return fmt.Sprintf("Path: %s, Name: %s, ID: %s, Carnet: %s",
		p.Path, p.Name, p.ID, p.Carnet)
}

type MkdirParams struct {
	Path string // Ruta de la carpeta a crear
	P    bool   // Si debe crear directorios padre
}

func (p *MkdirParams) String() string {
	return fmt.Sprintf("Path: %s, P: %t", p.Path, p.P)
}

type ViewUsersParams struct {
	ID string // ID de la partición montada
}

func (p *ViewUsersParams) String() string {
	return fmt.Sprintf("ID: %s", p.ID)
}

type PartdiskParams struct {
	Path string // Ruta del disco
}

func (p *PartdiskParams) String() string {
	return fmt.Sprintf("Path: %s", p.Path)
}

type LSParams struct {
	Path string // Ruta del disco
	Name string // Nombre de la partición
	Of   string // Ruta dentro de la partición
}

func (p *LSParams) String() string {
	return fmt.Sprintf("Path: %s, Name: %s, Of: %s", p.Path, p.Name, p.Of)
}

type UnmountParams struct {
	ID string // ID de la partición montada
}

func (p *UnmountParams) String() string {
	return fmt.Sprintf("ID: %s", p.ID)
}

type RemoveParams struct {
	Path string // Ruta del archivo o carpeta a eliminar
}

func (p *RemoveParams) String() string {
	return fmt.Sprintf("Path: %s", p.Path)
}

type MoveParams struct {
	Path    string // Ruta del archivo o carpeta a mover
	Destino string // Ruta de destino donde se moverá el archivo o carpeta
}

func (p *MoveParams) String() string {
	return fmt.Sprintf("Path: %s, Destino: %s", p.Path, p.Destino)
}

type FindParams struct {
	Path string // Ruta de la carpeta donde iniciar la búsqueda
	Name string // Nombre del archivo o carpeta a buscar (puede contener * y ?)
}

func (p *FindParams) String() string {
	return fmt.Sprintf("Path: %s, Name: %s", p.Path, p.Name)
}

type ChownParams struct {
	Path    string // Ruta del archivo o carpeta al que cambiar el propietario
	Usuario string // Nombre del nuevo propietario
	R       bool   // Recursivo para carpetas
}

func (p *ChownParams) String() string {
	return fmt.Sprintf("Path: %s, Usuario: %s, R: %t", p.Path, p.Usuario, p.R)
}

type ChmodParams struct {
	Path string // Ruta del archivo o carpeta al que cambiar los permisos
	UGO  string // Permisos en formato UGO (0-7)(0-7)(0-7)
	R    bool   // Recursivo para carpetas
}

func (p *ChmodParams) String() string {
	return fmt.Sprintf("Path: %s, UGO: %s, R: %t", p.Path, p.UGO, p.R)
}

// RecoveryParams representa los parámetros del comando RECOVERY
type RecoveryParams struct {
	ID string // ID de la partición
}

func (p *RecoveryParams) String() string {
	return fmt.Sprintf("ID: %s", p.ID)
}

// LossParams representa los parámetros del comando LOSS
type LossParams struct {
	ID string // ID de la partición
}

func (p *LossParams) String() string {
	return fmt.Sprintf("ID: %s", p.ID)
}

// JournalingParams representa los parámetros del comando JOURNALING
type JournalingParams struct {
	ID string // ID de la partición
}

func (p *JournalingParams) String() string {
	return fmt.Sprintf("ID: %s", p.ID)
}
