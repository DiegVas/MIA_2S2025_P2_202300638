# Proyecto No.1 Manejo e implementacion de archivos - Manual tecnico

![alt text](src/image.png)

- Nombre: Diego Alejandro Vásquez Alonzo
- Carne: 202300638

# Arquitectura del Sistema

## 1. Arquitectura General del Sistema

El sistema está dividido en dos módulos principales que trabajan en conjunto para simular un sistema de archivos EXT2:

- **Frontend**: Aplicación Vue.js que proporciona una interfaz de usuario tipo terminal
- **Backend**: Servidor HTTP en Go que procesa comandos y gestiona el sistema de archivos

### 1.1 Comunicación entre Módulos

La comunicación entre frontend y backend se realiza mediante **API REST** usando protocolo HTTP, con intercambio de datos en formato JSON.

**Flujo de comunicación:**

```
Usuario → Frontend (Vue.js) → HTTP Request → Backend (Go) → Filesystem Operations → HTTP Response → Frontend → Usuario
```

![alt text](<src/Diagrama archivos.png>)

## 2. Módulo Frontend (Vue.js + TypeScript)

### 2.1 Estructura del Frontend

```
frontend/
├── src/
│   ├── App.vue                    # Componente raíz
│   ├── main.ts                    # Punto de entrada de la aplicación
│   ├── components/
│   │   ├── layout/
│   │   │   ├── Header_Layout.vue  # Header de la aplicación
│   │   │   └── Body_Layout.vue    # Cuerpo principal con chat
│   │   └── common/
│   │       ├── ChatInput.vue      # Input para comandos
│   │       └── SmiaViewer.vue     # Visor de archivos .smia
│   ├── services/
│   │   └── send_commands.ts       # Servicio de comunicación con backend
│   ├── styles/                    # Estilos CSS
│   └── types/                     # Definiciones de tipos TypeScript
├── package.json                   # Dependencias y scripts
└── vite.config.ts                # Configuración de Vite
```

### 2.2 Componentes Clave del Frontend

#### 2.2.1 Body_Layout.vue - Interfaz Principal

```vue
<script setup>
import { sendCommand } from "../../services/send_commands.ts";

const handleSendMessage = async (message) => {
  // 1. Agregar mensaje del usuario a la interfaz
  messages.value.push({
    id: crypto.randomUUID(),
    text: message,
    timestamp: new Date(),
    type: "user",
  });

  // 2. Crear objeto de comando para enviar al backend
  const commandObj = {
    command: message,
    timestamp: new Date().toISOString(),
    adjunt_file: null,
  };

  // 3. Enviar comando al backend y recibir respuesta
  const system_response = await sendCommand(commandObj);

  // 4. Mostrar respuesta del sistema
  messages.value.push({
    id: crypto.randomUUID(),
    ...system_response,
  });
};
</script>
```

**Funciones principales:**

- **Gestión de mensajes**: Mantiene historial de comandos y respuestas
- **Interfaz tipo chat**: Simula una terminal con interfaz moderna
- **Manejo de archivos .smia**: Permite cargar y ejecutar scripts de comandos
- **Categorización de respuestas**: Diferencia entre mensajes de usuario, sistema y errores

#### 2.2.2 send_commands.ts - Servicio de Comunicación

```typescript
export async function sendCommand(
  command: SendCommand
): Promise<{ type: string; text: string; timestamp: Date } | undefined> {
  try {
    const response = await fetch("http://localhost:8080/command", {
      method: "POST",
      body: JSON.stringify(command),
      headers: {
        "Content-Type": "application/json",
      },
    });

    const data = await response.json();

    if (response.ok) {
      return {
        type: data.type,
        text: data.message,
        timestamp: new Date(),
      };
    } else {
      return {
        type: data.type || "system-error",
        text: data.message || "Error al enviar el comando",
        timestamp: new Date(),
      };
    }
  } catch (error) {
    // Manejo de errores de red
  }
}
```

**Características:**

- **Comunicación HTTP**: Realiza peticiones POST al endpoint `/command`
- **Manejo de errores**: Gestiona errores de red y respuestas HTTP
- **Tipado fuerte**: Utiliza TypeScript para validación de tipos
- **Configuración CORS**: Permite comunicación cross-origin

### 2.3 Tecnologías del Frontend

- **Vue.js 3**: Framework reactivo con Composition API
- **TypeScript**: Tipado estático para mayor robustez
- **Vite**: Herramienta de desarrollo y construcción
- **Tailwind CSS**: Framework de utilidades CSS para diseño

## 3. Módulo Backend (Go)

### 3.1 Estructura del Backend

```
backend/
├── main.go                    # Punto de entrada del servidor
├── routes/                    # Manejadores de rutas HTTP
│   ├── handlePostUser.go      # Router principal de comandos
│   ├── handleMkdisk.go        # Creación de discos
│   ├── handleFdisk.go         # Particionado
│   ├── handleMount.go         # Montaje de particiones
│   ├── handleMkfs.go          # Formateo EXT2
│   ├── handleLogin.go         # Autenticación
│   ├── handleUser.go          # Gestión de usuarios
│   ├── handleGroup.go         # Gestión de grupos
│   ├── handleMkdir.go         # Creación de directorios
│   ├── handleMkfile.go        # Creación de archivos
│   ├── handleCat.go           # Lectura de archivos
│   └── handleRep.go           # Generación de reportes
├── Analyzer/                  # Parsers de comandos
├── provider/                  # Lógica de negocio
├── structs/                   # Estructuras de datos
└── utils/                     # Utilidades (CORS, respuestas)
```

### 3.2 Componentes Clave del Backend

#### 3.2.1 main.go - Servidor HTTP

```go
func main() {
    // Configurar endpoint principal
    http.HandleFunc("/command", utils.CORSMiddleware(routes.HandlePostUser))

    // Iniciar servidor en puerto 8080
    fmt.Println("Servidor iniciado en http://localhost:8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

**Funciones:**

- **Servidor HTTP**: Escucha en puerto 8080
- **Middleware CORS**: Permite comunicación cross-origin
- **Punto de entrada único**: Todos los comandos pasan por `/command`

#### 3.2.2 handlePostUser.go - Router de Comandos

```go
func HandlePostUser(w http.ResponseWriter, r *http.Request) {
    // 1. Validar método HTTP
    if r.Method != http.MethodPost {
        http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
        return
    }

    // 2. Leer y parsear el cuerpo de la petición
    body, err := io.ReadAll(r.Body)
    var command structs.Command
    err = json.Unmarshal(body, &command)

    // 3. Determinar tipo de comando y rutear
    commandLower := strings.ToLower(command.Command)
    commandBase := strings.Split(commandLower, " ")[0]

    switch commandBase {
    case "mkdisk":
        handleMkdisk(w, command.Command)
    case "fdisk":
        handleFdisk(w, command.Command)
    case "mount":
        handleMount(w, command.Command)
    case "mkfs":
        handleMkfs(w, command.Command)
    case "login":
        handleLogin(w, command.Command)
    // ... más comandos
    }
}
```

**Funciones principales:**

- **Routing centralizado**: Distribuye comandos a handlers específicos
- **Validación de entrada**: Verifica formato JSON y método HTTP
- **Parseo de comandos**: Identifica el tipo de comando a ejecutar

#### 3.2.3 Handlers Específicos

**Ejemplo: handleMkdisk.go**

```go
func handleMkdisk(w http.ResponseWriter, command string) {
    // 1. Parsear parámetros del comando
    params, err := Analyzer.ParseMkdisk(command)
    if err != nil {
        response := structs.Response{
            Message: fmt.Sprintf("Error en el comando: %s", err.Error()),
            Type:    "system-error",
        }
        utils.SendJSONResponse(w, response, http.StatusBadRequest)
        return
    }

    // 2. Ejecutar lógica de negocio
    err = utils.CreateDisk(params)
    if err != nil {
        // Manejo de errores
        return
    }

    // 3. Enviar respuesta exitosa
    response := structs.Response{
        Message: fmt.Sprintf("Disco creado exitosamente: %s", params.Path),
        Data:    params,
        Type:    "system",
    }
    utils.SendJSONResponse(w, response, http.StatusCreated)
}
```

### 3.3 Arquitectura por Capas del Backend

#### 3.3.1 Capa de Rutas (Routes)

- **Responsabilidad**: Manejo de peticiones HTTP
- **Funciones**: Validación de entrada, routing, respuestas HTTP
- **Ubicación**: `/routes/`

#### 3.3.2 Capa de Análisis (Analyzer)

- **Responsabilidad**: Parseo y validación de comandos
- **Funciones**: Extraer parámetros, validar sintaxis
- **Ubicación**: `/Analyzer/`

#### 3.3.3 Capa de Proveedores (Provider)

- **Responsabilidad**: Lógica de negocio del sistema de archivos
- **Módulos principales**:
  - `mounted/`: Gestión de particiones montadas
  - `files/`: Operaciones con archivos
  - `users/`: Gestión de usuarios y grupos
  - `reports/`: Generación de reportes

#### 3.3.4 Capa de Utilidades (Utils)

- **Responsabilidad**: Funciones auxiliares
- **Funciones**: CORS, respuestas JSON, operaciones de disco

## 4. Protocolo de Comunicación

### 4.1 Formato de Petición

```json
{
  "command": "mkdisk -size=10 -path=/home/user/disk1.mia",
  "timestamp": "2025-09-12T10:30:00Z",
  "adjunt_file": null
}
```

### 4.2 Formato de Respuesta

```json
{
  "message": "Disco creado exitosamente: /home/user/disk1.mia",
  "data": {
    "Size": 10,
    "Path": "/home/user/disk1.mia",
    "Unit": "M",
    "Fit": "FF"
  },
  "type": "system"
}
```

### 4.3 Tipos de Respuesta

- **`system`**: Operaciones exitosas del sistema
- **`system-error`**: Errores en comandos o parámetros
- **`auth-success`**: Autenticación exitosa
- **`auth-error`**: Errores de autenticación
- **`mounted-partitions`**: Información de particiones montadas

## 5. Gestión de Estados y Sesiones

### 5.1 Estado del Frontend

- **Mensajes**: Historial de comandos y respuestas
- **UI State**: Estado de componentes y animaciones
- **File State**: Contenido de archivos .smia cargados

### 5.2 Estado del Backend

- **Particiones montadas**: Mapa en memoria de particiones activas
- **Sesión de usuario**: Estado de autenticación (singleton)
- **Archivos binarios**: Gestión de archivos .mia en disco

## 6. Flujo de Operaciones EXT2

### 6.1 Creación y Formateo de Disco

```
Frontend: "mkdisk -size=10 -path=/disk.mia"
    ↓
Backend: Parseo → Validación → Creación archivo binario → Inicialización MBR
    ↓
Frontend: Confirmación de creación

Frontend: "fdisk -size=5 -path=/disk.mia -name=part1"
    ↓
Backend: Lectura MBR → Cálculo espacio → Creación partición → Actualización MBR
    ↓
Frontend: Confirmación de partición

Frontend: "mount -path=/disk.mia -name=part1"
    ↓
Backend: Validación partición → Registro en memoria → Generación ID
    ↓
Frontend: ID de montaje

Frontend: "mkfs -id=382part1"
    ↓
Backend: Inicialización superbloque → Creación bitmaps → Tabla inodos → Archivo users.txt
    ↓
Frontend: Confirmación de formateo EXT2
```

### 6.2 Operaciones de Archivos

```
Frontend: "login -user=root -pass=123 -id=382part1"
    ↓
Backend: Validación credenciales → Actualización sesión → Permisos
    ↓
Frontend: Sesión iniciada

Frontend: "mkdir -path=/home -id=382part1"
    ↓
Backend: Verificación permisos → Búsqueda inodo padre → Creación inodo directorio → Actualización estructura
    ↓
Frontend: Directorio creado

Frontend: "mkfile -path=/home/test.txt -size=100"
    ↓
Backend: Creación inodo archivo → Asignación bloques → Escritura contenido
    ↓
Frontend: Archivo creado
```

## 7. Manejo de Errores y Validaciones

### 7.1 Validaciones del Frontend

- **Formato de entrada**: Verificación de comandos válidos
- **Estado de conexión**: Manejo de errores de red
- **Tipos de archivo**: Validación de archivos .smia

### 7.2 Validaciones del Backend

- **Sintaxis de comandos**: Parseo y validación de parámetros
- **Estado del sistema**: Verificación de particiones montadas y sesiones
- **Integridad de datos**: Validación de estructuras EXT2
- **Permisos**: Control de acceso a archivos y directorios

## 8. Configuración CORS

### 8.1 Configuración del Backend

```go
func EnableCORS(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
    w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
    w.Header().Set("Access-Control-Allow-Credentials", "true")
}
```

### 8.2 Middleware CORS

```go
func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        EnableCORS(w, r)
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        next(w, r)
    }
}
```

# Descripción de Estructuras de Datos Fundamentales del Sistema de Archivos

Este documento describe las principales estructuras de datos utilizadas en el sistema de archivos, organizadas y gestionadas dentro de los archivos binarios `.mia`. Estas estructuras proporcionan la base para la gestión de discos, particiones, sistema de archivos EXT2 e información de usuarios y grupos.

## 1. Master Boot Record (MBR) y Particiones

### 1.1 Master Boot Record (MBR)

```go
type MBR struct {
    Mbr_size       int32         // Tamaño del disco en bytes
    Mbr_timestamp  [10]byte      // Fecha de creación del disco
    Mbr_uuid       [16]byte      // Identificador único del disco
    Mbr_partitions int32         // Número de particiones en el disco
    Partitions     [4]PARTITION  // Tabla de particiones (máximo 4)
    Dsk_fit        [1]byte       // Algoritmo de ajuste del disco (BF, FF, WF)
}
```

**Función dentro del sistema:**

- Es la primera estructura de control del disco, ubicada al inicio del archivo `.mia`
- Contiene metadatos del disco completo y la tabla de particiones primarias
- Permite la identificación única del disco mediante UUID
- Define el algoritmo de ajuste para la asignación de espacios en el disco

**Organización en el archivo binario:**

- Se encuentra en los primeros bytes del archivo `.mia`
- Tamaño fijo de la estructura que incluye espacio para 4 particiones

### 1.2 Estructura de Partición (PARTITION)

```go
type PARTITION struct {
    Part_status     [1]byte   // Estado de la partición (activa/inactiva)
    Part_type       [1]byte   // Tipo: P(rimaria), E(xtendida), L(ógica)
    Part_fit        [1]byte   // Algoritmo de ajuste (BF, FF, WF)
    Part_start      int32     // Sector donde inicia la partición
    Part_size       int32     // Tamaño de la partición en bytes
    Part_name       [16]byte  // Nombre de la partición
    Part_uuid       [16]byte  // Identificador único de la partición
    Part_correlative int32    // Número correlativo de la partición
}
```

**Función dentro del sistema:**

- Define las características y ubicación de cada partición en el disco
- Controla el espacio asignado y el tipo de partición
- Facilita la identificación única de cada partición mediante UUID

### 1.3 Extended Boot Record (EBR)

```go
type EBR struct {
    Part_mount  [1]byte    // Indica si la partición está montada
    Part_fit    [1]byte    // Algoritmo de ajuste
    Part_start  int32      // Sector de inicio de la partición lógica
    Part_size   int32      // Tamaño de la partición lógica
    Part_next   int32      // Puntero al siguiente EBR (-1 si es el último)
    Part_name   [16]byte   // Nombre de la partición lógica
}
```

**Función dentro del sistema:**

- Gestiona las particiones lógicas dentro de una partición extendida
- Forma una lista enlazada de particiones lógicas
- Permite extender el número de particiones más allá del límite de 4 particiones primarias

**Organización en el archivo binario:**

- Se ubican inmediatamente después del inicio de cada partición lógica
- Forman una cadena enlazada donde cada EBR apunta al siguiente

## 2. Sistema de Archivos EXT2

### 2.1 Superbloque (Superblock)

```go
type Superblock struct {
    S_filesystem_type   int32    // Tipo de sistema de archivos (2=ext2, 3=ext3)
    S_inodes_count      int32    // Número total de inodos
    S_blocks_count      int32    // Número total de bloques
    S_free_blocks_count int32    // Número de bloques libres
    S_free_inodes_count int32    // Número de inodos libres
    S_mtime             [16]byte // Última fecha de montaje
    S_umtime            [16]byte // Última fecha de desmontaje
    S_mnt_count         int32    // Contador de montajes
    S_magic             int32    // Número mágico (0xEF53)
    S_inode_size        int32    // Tamaño de cada inodo
    S_block_size        int32    // Tamaño de cada bloque
    S_first_ino         int32    // Primer inodo libre
    S_first_blo         int32    // Primer bloque libre
    S_bm_inode_start    int32    // Inicio del bitmap de inodos
    S_bm_block_start    int32    // Inicio del bitmap de bloques
    S_inode_start       int32    // Inicio de la tabla de inodos
    S_block_start       int32    // Inicio de la tabla de bloques
}
```

**Función dentro del sistema:**

- Es la estructura de control principal del sistema de archivos EXT2
- Contiene información crítica sobre la organización y estado del sistema de archivos
- Permite validar la integridad mediante el número mágico
- Facilita la localización de todas las demás estructuras del sistema de archivos

**Organización en el archivo binario:**

- Se ubica al inicio de cada partición formateada con EXT2
- Tamaño: 88 bytes

### 2.2 Inodos (Inode)

```go
type Inode struct {
    I_uid   int32     // ID del usuario propietario
    I_gid   int32     // ID del grupo propietario
    I_size  int32     // Tamaño del archivo/directorio en bytes
    I_atime [16]byte  // Última fecha de acceso
    I_ctime [16]byte  // Fecha de creación
    I_mtime [16]byte  // Última fecha de modificación
    I_block [15]int32 // Punteros a bloques (12 directos + 3 indirectos)
    I_type  int32     // Tipo: 0=archivo, 1=directorio
    I_perm  int32     // Permisos del archivo/directorio
}
```

**Función dentro del sistema:**

- Es la estructura fundamental que representa archivos y directorios
- Contiene metadatos completos de cada elemento del sistema de archivos
- Gestiona la asignación de bloques mediante punteros directos e indirectos
- Controla permisos y propiedad de archivos y directorios

**Organización en el archivo binario:**

- Se almacenan en la tabla de inodos, cuya ubicación es definida por el superbloque
- Tamaño: 128 bytes por inodo
- El inodo 0 corresponde al directorio raíz

### 2.3 Bloques de Datos

#### 2.3.1 Bloque de Directorio (FolderBlock)

```go
type FolderBlock struct {
    B_content [4]Content // Hasta 4 entradas por bloque de directorio
}

type Content struct {
    B_name  [12]byte // Nombre del archivo/directorio (máximo 12 caracteres)
    B_inodo int32    // Número de inodo correspondiente
}
```

**Función dentro del sistema:**

- Almacena las entradas de un directorio
- Relaciona nombres de archivos/directorios con sus respectivos inodos
- Permite la navegación y búsqueda dentro del sistema de archivos

#### 2.3.2 Bloque de Archivo (FileBlock)

```go
type FileBlock struct {
    B_content [64]byte // Contenido del archivo (64 bytes por bloque)
}
```

**Función dentro del sistema:**

- Almacena el contenido real de los archivos
- Proporciona 64 bytes de espacio útil por bloque
- Se enlaza mediante los punteros en los inodos

#### 2.3.3 Bloque de Punteros (PointerBlock)

```go
type PointerBlock struct {
    B_pointers [16]int32 // 16 punteros a otros bloques
}
```

**Función dentro del sistema:**

- Implementa el direccionamiento indirecto para archivos grandes
- Permite extender el espacio de direccionamiento más allá de los 12 punteros directos del inodo
- Soporta hasta 3 niveles de indirección (simple, doble, triple)

**Organización en el archivo binario:**

- Todos los bloques tienen un tamaño fijo de 1024 bytes
- Se ubican en la región de bloques definida por el superbloque
- Los bloques se gestionan mediante bitmaps para control de espacio libre

## 3. Gestión de Usuarios y Grupos

### 3.1 Estructura de Usuario

```go
type User struct {
    Username string // Nombre de usuario
    Password string // Contraseña (puede estar encriptada)
    Group    string // Grupo al que pertenece
    UID      int32   // ID único de usuario
    GID      int32   // ID del grupo
}
```

**Función dentro del sistema:**

- Define los usuarios del sistema de archivos
- Controla el acceso y permisos mediante UID/GID
- Se almacena en el archivo `users.txt` dentro del sistema de archivos

### 3.2 Parámetros de Usuario y Grupo

```go
type UserParams struct {
    User string // Nombre de usuario
    Pass string // Contraseña
    Grp  string // Grupo
}

type GroupParams struct {
    Name string // Nombre del grupo
}
```

**Función dentro del sistema:**

- Estructuras auxiliares para la gestión de usuarios y grupos
- Facilitan las operaciones de creación y modificación de usuarios
- Permiten la validación de parámetros antes de la persistencia

## 4. Gestión de Sesiones

### 4.1 Sesión de Usuario

```go
type Session struct {
    Active bool   // Estado de la sesión
    User   string // Usuario activo
    ID     string // ID de la partición montada
    Group  string // Grupo del usuario
}
```

**Función dentro del sistema:**

- Mantiene el estado de autenticación del usuario activo
- Controla qué partición está siendo utilizada
- Implementa patrón Singleton para garantizar una sola sesión activa
- Facilita el control de permisos en operaciones del sistema de archivos

## 5. Constantes del Sistema

```go
const (
    SUPERBLOCK_SIZE   = 88    // Tamaño del superbloque
    INODE_SIZE        = 128   // Tamaño de un inodo
    BLOCK_SIZE        = 1024  // Tamaño de un bloque
    FILE_CONTENT_SIZE = 64    // Tamaño útil para contenido de archivos
    FOLDER_SIZE       = 64    // Tamaño de una carpeta
    POINTER_SIZE      = 64    // Tamaño de un bloque de punteros
    EXT2_MAGIC        = 0xEF53 // Número mágico para validación EXT2
    ROOT_INODE        = 0      // Número del inodo raíz
)
```

## 6. Organización General en el Archivo .mia

La organización típica de un archivo `.mia` es la siguiente:

1. **Sector 0**: MBR (Master Boot Record)
2. **Sectores siguientes**: Particiones según la tabla del MBR
3. **Por cada partición EXT2**:
   - Superbloque
   - Bitmap de inodos
   - Bitmap de bloques
   - Tabla de inodos
   - Tabla de bloques de datos

Esta organización permite un acceso eficiente y estructurado a toda la información del sistema de archivos, manteniendo la integridad de los datos y facilitando las operaciones de lectura, escritura y navegación dentro del sistema.

Las estructuras están diseñadas para optimizar el espacio de almacenamiento mientras proporcionan toda la funcionalidad necesaria para un sistema de archivos completo, incluyendo gestión de permisos, usuarios, grupos y operaciones avanzadas de archivos y directorios.

# MIA Lista de comandos permitidos

## Tabla de Contenidos

1. [Administración de Discos](#administración-de-discos)
2. [Administración del Sistema de Archivos](#administración-del-sistema-de-archivos)
3. [Administración de Usuarios y Grupos](#administración-de-usuarios-y-grupos)
4. [Administración de Carpetas, Archivos y Permisos](#administración-de-carpetas-archivos-y-permisos)
5. [Reportes](#reportes)

---

## Administración de Discos

### 1. MKDISK

**Propósito**: Crea un archivo binario que simula un disco duro virtual con extensión `.mia`.

**Parámetros**:

- `-size` (Obligatorio): Tamaño del disco a crear (debe ser positivo y mayor que cero)
- `-fit` (Opcional): Tipo de ajuste para particiones
  - `BF`: Best Fit (Mejor Ajuste)
  - `FF`: First Fit (Primer Ajuste) - _valor por defecto_
  - `WF`: Worst Fit (Peor Ajuste)
- `-unit` (Opcional): Unidades del parámetro size
  - `K`: Kilobytes (1024 bytes)
  - `M`: Megabytes (1024 * 1024 bytes) - *valor por defecto\*
- `-path` (Obligatorio): Ruta donde se creará el archivo del disco

**Efectos en las Estructuras**:

- Crea un archivo binario inicializado con ceros
- Escribe la estructura MBR en el primer sector
- Inicializa las 4 particiones con valores por defecto
- Establece el tamaño total del disco

**Ejemplos**:

```bash
# Crea un disco de 3000 KB
mkdisk -Size=3000 -unit=K -path=/home/user/Disco1.mia

# Crea un disco de 5 MB con Best Fit
mkdisk -size=5 -unit=M -fit=BF -path="/home/mis discos/Disco3.mia"

# Crea un disco de 10 MB (unit por defecto)
mkdisk -size=10 -path="/home/mis discos/Disco5.mia"
```

### 2. RMDISK

**Propósito**: Elimina un archivo que representa un disco duro virtual.

**Parámetros**:

- `-path` (Obligatorio): Ruta del archivo de disco a eliminar

**Efectos en las Estructuras**:

- Elimina físicamente el archivo `.mia` del sistema
- Libera el espacio ocupado en el disco físico

**Ejemplo**:

```bash
# Elimina Disco4.mia
rmdisk -path="/home/mis discos/Disco4.mia"
```

### 3. FDISK

**Propósito**: Administra particiones en el disco virtual (crear, eliminar, modificar).

**Parámetros**:

- `-size` (Obligatorio para crear): Tamaño de la partición (positivo y mayor que cero)
- `-unit` (Opcional): Unidades del parámetro size
  - `B`: Bytes
  - `K`: Kilobytes - _valor por defecto_
  - `M`: Megabytes
- `-path` (Obligatorio): Ruta del disco donde se creará la partición
- `-type` (Opcional): Tipo de partición
  - `P`: Partición Primaria - _valor por defecto_
  - `E`: Partición Extendida
  - `L`: Partición Lógica
- `-fit` (Opcional): Tipo de ajuste
  - `BF`: Best Fit
  - `FF`: First Fit
  - `WF`: Worst Fit - _valor por defecto_
- `-name` (Obligatorio): Nombre de la partición

**Restricciones**:

- Máximo 4 particiones primarias + extendidas
- Solo una partición extendida por disco
- Particiones lógicas solo dentro de extendida
- Nombres únicos por disco

**Efectos en las Estructuras**:

- **Particiones Primarias**: Actualiza el MBR, reserva espacio en el disco
- **Particiones Extendidas**: Actualiza el MBR, crea el primer EBR
- **Particiones Lógicas**: Crea nuevos EBRs enlazados, actualiza la cadena de EBRs

**Ejemplos**:

```bash
# Crea partición primaria de 300KB
fdisk -Size=300 -path=/home/Disco1.mia -name=Particion1

# Crea partición extendida
fdisk -type=E -path=/home/Disco2.mia -Unit=K -name=Particion2 -size=300

# Crea partición lógica con Best Fit
fdisk -size=1 -type=L -unit=M -fit=BF -path="/mis discos/Disco3.mia" -name="Particion3"
```

### 4. MOUNT

**Propósito**: Monta una partición del disco en el sistema, asignándole un ID único.

**Parámetros**:

- `-path` (Obligatorio): Ruta del disco que contiene la partición
- `-name` (Obligatorio): Nombre de la partición a montar

**Sistema de IDs**:
Formato: `[Últimos 2 dígitos del carnet][Número de partición][Letra del disco]`

- Ejemplo con carnet 202401234: 341A, 341B, 341C, 342A, 343A

**Efectos en las Estructuras**:

- Actualiza el campo `part_status` a montado
- Asigna el correlativo de partición
- Genera y asigna el ID único
- Mantiene registro en memoria RAM (no en disco)

**Ejemplos**:

```bash
mount -path=/home/Disco2.mia -name=Part2 # ID: 341A
mount -path=/home/Disco2.mia -name=Part1 # ID: 342A
mount -path=/home/Disco1.mia -name=Part3 # ID: 341B
```

### 5. MOUNTED

**Propósito**: Muestra todas las particiones montadas en memoria.

**Parámetros**: Ninguno

**Efectos en las Estructuras**:

- Solo lectura, no modifica estructuras
- Consulta el registro en memoria RAM

**Ejemplo**:

```bash
mounted
# Salida esperada: 341A, 342A, 341B, 341C
```

---

## Administración del Sistema de Archivos

### 6. MKFS

**Propósito**: Formatea una partición montada con el sistema de archivos EXT2.

**Parámetros**:

- `-id` (Obligatorio): ID de la partición montada generado con MOUNT
- `-type` (Opcional): Tipo de formateo
  - `full`: Formateo completo - _valor por defecto_

**Efectos en las Estructuras**:

- Crea el **Superbloque** con metadatos del sistema EXT2
- Inicializa **Bitmap de Inodos** (todos en 0, usables)
- Inicializa **Bitmap de Bloques** (todos en 0, usables)
- Crea la **Tabla de Inodos** vacía
- Crea la **Tabla de Bloques** vacía
- Crea el **archivo users.txt** en la raíz con usuario root por defecto
- Establece los punteros de inicio para todas las estructuras

**Cálculo de Estructuras**:

```
tamaño_particion = sizeof(superblock) + n + 3*n + n*sizeof(inodo) + 3*n*sizeof(block)
numero_estructuras = floor(n)
```

**Ejemplos**:

```bash
# Formateo completo
mkfs -type=full -id=341A

# Formateo por defecto
mkfs -id=342A
```

### 7. CAT

**Propósito**: Muestra el contenido de uno o varios archivos concatenados.

**Parámetros**:

- `-filen` (Obligatorio): Lista de archivos a mostrar (file1, file2, file3, etc.)

**Validaciones**:

- Verifica permisos de lectura del usuario actual
- Valida existencia de archivos
- Requiere sesión activa

**Efectos en las Estructuras**:

- **Lectura**: Accede a los inodos de los archivos
- **Lectura**: Sigue los punteros directos e indirectos
- **Lectura**: Lee el contenido de los bloques de archivo
- Actualiza `i_atime` (último acceso) en los inodos

**Ejemplos**:

```bash
# Lee un archivo
cat -file1=/home/user/docs/a.txt

# Concatena múltiples archivos
cat -file1="/home/a.txt" -file2="/home/b.txt" -file3="/home/c.txt"
```

---

## Administración de Usuarios y Grupos

### 8. LOGIN

**Propósito**: Inicia sesión en el sistema con un usuario específico.

**Parámetros**:

- `-user` (Obligatorio): Nombre de usuario (case-sensitive)
- `-pass` (Obligatorio): Contraseña del usuario (case-sensitive)
- `-id` (Obligatorio): ID de la partición montada donde iniciar sesión

**Efectos en las Estructuras**:

- **Lectura**: Accede al archivo `users.txt` para validar credenciales
- Establece la sesión activa en memoria
- Define el contexto de trabajo (partición y permisos)

**Restricciones**:

- Solo una sesión activa a la vez
- Debe usar LOGOUT antes de otro LOGIN

**Ejemplo**:

```bash
login -user=root -pass=123 -id=062A
```

### 9. LOGOUT

**Propósito**: Cierra la sesión activa del usuario.

**Parámetros**: Ninguno

**Efectos en las Estructuras**:

- Limpia la sesión de memoria
- No modifica estructuras del disco

**Ejemplo**:

```bash
logout
```

### 10. MKGRP

**Propósito**: Crea un nuevo grupo de usuarios (solo usuario root).

**Parámetros**:

- `-name` (Obligatorio): Nombre del grupo (case-sensitive)

**Efectos en las Estructuras**:

- **Escritura**: Modifica el archivo `users.txt`
- Agrega nueva línea: `GID, G, nombre_grupo`
- Incrementa el contador de GID
- Actualiza los bitmaps e inodos del archivo users.txt

**Formato en users.txt**:

```
2, G, usuarios
```

**Ejemplo**:

```bash
mkgrp -name=usuarios
```

### 11. RMGRP

**Propósito**: Elimina un grupo existente (solo usuario root).

**Parámetros**:

- `-name` (Obligatorio): Nombre del grupo a eliminar

**Efectos en las Estructuras**:

- **Escritura**: Modifica el archivo `users.txt`
- Cambia el GID a 0 (marca como eliminado)
- Actualiza los bloques del archivo users.txt

**Formato en users.txt después de eliminar**:

```
0, G, usuarios
```

**Ejemplo**:

```bash
rmgrp -name=usuarios
```

### 12. MKUSR

**Propósito**: Crea un nuevo usuario (solo usuario root).

**Parámetros**:

- `-user` (Obligatorio): Nombre del usuario (máximo 10 caracteres)
- `-pass` (Obligatorio): Contraseña (máximo 10 caracteres)
- `-grp` (Obligatorio): Grupo al que pertenece (debe existir)

**Efectos en las Estructuras**:

- **Escritura**: Modifica el archivo `users.txt`
- Agrega nueva línea: `UID, U, grupo, usuario, contraseña`
- Incrementa el contador de UID
- Valida existencia del grupo

**Formato en users.txt**:

```
2, U, usuarios, user1, usuario
```

**Ejemplo**:

```bash
mkusr -user=user1 -pass=usuario -grp=usuarios
```

### 13. RMUSR

**Propósito**: Elimina un usuario existente (solo usuario root).

**Parámetros**:

- `-user` (Obligatorio): Nombre del usuario a eliminar

**Efectos en las Estructuras**:

- **Escritura**: Modifica el archivo `users.txt`
- Cambia el UID a 0 (marca como eliminado)

**Ejemplo**:

```bash
rmusr -user=user1
```

### 14. CHGRP

**Propósito**: Cambia el grupo de un usuario existente (solo usuario root).

**Parámetros**:

- `-user` (Obligatorio): Usuario al que cambiar el grupo
- `-grp` (Obligatorio): Nuevo grupo (debe existir)

**Efectos en las Estructuras**:

- **Escritura**: Modifica el archivo `users.txt`
- Actualiza la línea del usuario con el nuevo grupo

**Ejemplo**:

```bash
chgrp -user=user2 -grp=grupo1
```

---

## Administración de Carpetas, Archivos y Permisos

### 15. MKFILE

**Propósito**: Crea un archivo nuevo en el sistema de archivos.

**Parámetros**:

- `-path` (Obligatorio): Ruta completa del archivo a crear
- `-r` (Opcional): Crea carpetas padres si no existen
- `-size` (Opcional): Tamaño en bytes (contenido: números 0-9 repetidos)
- `-cont` (Opcional): Ruta a archivo local para copiar contenido

**Permisos por Defecto**: 664 (rw-rw-r--)

**Efectos en las Estructuras**:

- **Creación de Inodo**: Nuevo inodo con `I_type=0` (archivo)
- **Asignación de Bloques**: Según el tamaño del archivo
- **Actualización de Bitmap Inodos**: Marca el inodo como ocupado (1)
- **Actualización de Bitmap Bloques**: Marca los bloques usados (1)
- **Modificación de Carpeta Padre**: Agrega entrada en bloque de carpeta
- **Punteros Directos/Indirectos**: Según el tamaño del archivo

**Manejo de Bloques por Tamaño**:

- 0-764 bytes: Solo punteros directos (12 bloques)
- 765-16148 bytes: Incluye puntero indirecto simple
- Más bytes: Punteros dobles y triples indirectos

**Ejemplos**:

```bash
# Archivo con tamaño específico
mkfile -size=15 -path=/home/user/docs/a.txt -r

# Archivo con contenido de archivo local
mkfile -path=/home/user/docs/b.txt -r -cont=/home/Documents/b.txt

# Archivo vacío
mkfile -path="/home/mis documentos/archivo 1.txt"
```

### 16. MKDIR

**Propósito**: Crea una carpeta nueva en el sistema de archivos.

**Parámetros**:

- `-path` (Obligatorio): Ruta completa de la carpeta a crear
- `-p` (Opcional): Crea carpetas padres si no existen

**Permisos por Defecto**: 664 (rw-rw-r--)

**Efectos en las Estructuras**:

- **Creación de Inodo**: Nuevo inodo con `I_type=1` (carpeta)
- **Creación de Bloque Carpeta**: Con entradas "." y ".."
- **Actualización de Bitmaps**: Marca inodo y bloque como ocupados
- **Modificación de Carpeta Padre**: Agrega entrada para la nueva carpeta

**Estructura del Bloque Carpeta**:

```
Entrada 1: "." -> inodo actual
Entrada 2: ".." -> inodo padre
Entrada 3: (vacía)
Entrada 4: (vacía)
```

**Ejemplos**:

```bash
# Crear carpeta con padres
mkdir -p -path=/home/user/docs/usac

# Crear carpeta simple
mkdir -path="/home/mis documentos/archivos clases"
```

---

## Reportes

### 17. REP

**Propósito**: Genera reportes visuales del sistema de archivos usando Graphviz.

**Parámetros**:

- `-name` (Obligatorio): Tipo de reporte a generar
- `-path` (Obligatorio): Ruta y nombre del archivo de salida
- `-id` (Obligatorio): ID de la partición a reportar
- `-path_file_ls` (Opcional): Para reportes `file` y `ls`

**Tipos de Reportes Disponibles**:

#### MBR

Muestra información detallada del Master Boot Record y EBRs.

```bash
rep -id=A118 -path=/home/user/reports/reporte1.jpg -name=mbr
```

#### DISK

Visualiza la estructura de particiones y porcentajes de uso.

```bash
rep -id=A118 -path=/home/user/reports/report2.pdf -name=disk
```

#### INODE

Muestra todos los inodos utilizados con su información completa.

```bash
rep -id=A118 -path=/home/user/reports/report3.jpg -name=inode
```

#### BLOCK

Muestra información de todos los bloques utilizados.

```bash
rep -id=A118 -path=/home/user/reports/report4.jpg -name=block
```

#### BM_INODE

Genera bitmap de inodos en formato texto (20 registros por línea).

```bash
rep -id=A118 -path=/home/user/reports/report5.txt -name=bm_inode
```

#### BM_BLOCK

Genera bitmap de bloques en formato texto (20 registros por línea).

```bash
rep -id=A118 -path=/home/user/reports/report6.txt -name=bm_block
```

#### TREE

Genera árbol completo del sistema de archivos EXT2.

```bash
rep -id=A118 -path=/home/user/reports/report7.jpg -name=tree
```

#### SB

Muestra información del superbloque en formato tabla.

```bash
rep -id=A118 -path=/home/user/reports/report8.jpg -name=sb
```

#### FILE

Muestra nombre y contenido completo de un archivo específico.

```bash
rep -id=A118 -path=/home/user/reports/report9.txt -path_file_ls=/home/a.txt -name=file
```

#### LS

Lista archivos y carpetas con permisos, propietarios, fechas y tipos.

```bash
rep -id=A118 -path=/home/user/reports/report10.jpg -path_file_ls=/ -name=ls
```

---

## Nuevos Comandos del Sistema - Fase 2

### 18. LS

**Propósito**: Lista el contenido de un directorio mostrando archivos y subdirectorios.

**Parámetros**:

- `-path` (Opcional): Ruta del directorio a listar (por defecto: directorio actual)

**Efectos en las Estructuras**:

- **Lectura**: Accede al inodo del directorio especificado
- **Lectura**: Lee los bloques de directorio y sus entradas
- **Lectura**: Obtiene información de metadatos de cada archivo/directorio

**Ejemplo**:

```bash
ls -path=/home/user/docs
ls  # Lista directorio actual
```

### 19. EDIT

**Propósito**: Edita el contenido de un archivo existente con contenido desde un archivo local.

**Parámetros**:

- `-path` (Obligatorio): Ruta del archivo a editar en el sistema de archivos
- `-contenido` (Obligatorio): Ruta del archivo local que contiene el nuevo contenido

**Efectos en las Estructuras**:

- **Lectura/Escritura**: Accede al inodo del archivo objetivo
- **Liberación**: Libera todos los bloques actuales del archivo
- **Asignación**: Asigna nuevos bloques según el tamaño del nuevo contenido
- **Actualización**: Modifica el `i_size` y `i_mtime` del inodo
- **Journaling**: Registra la operación en el journal EXT3

**Ejemplo**:

```bash
edit -path=/home/user/documento.txt -contenido=/home/local/nuevo_contenido.txt
```

### 20. MOVE

**Propósito**: Mueve un archivo o directorio de una ubicación a otra dentro del sistema de archivos.

**Parámetros**:

- `-path` (Obligatorio): Ruta actual del archivo/directorio
- `-destino` (Obligatorio): Nueva ubicación destino

**Efectos en las Estructuras**:

- **Modificación**: Agrega entrada en el directorio destino
- **Eliminación**: Remueve entrada del directorio origen
- **Preservación**: Mantiene el mismo inodo (no copia datos)
- **Journaling**: Registra la operación de movimiento

**Validaciones**:

- Verifica permisos de escritura en origen y destino
- Valida que no exista conflicto de nombres en destino
- Requiere sesión activa

**Ejemplo**:

```bash
move -path=/home/user/archivo.txt -destino=/home/documents/
move -path=/home/folder1 -destino=/home/backup/folder1_backup
```

### 21. COPY

**Propósito**: Copia un archivo o directorio a otra ubicación, creando una referencia al mismo inodo.

**Parámetros**:

- `-path` (Obligatorio): Ruta del archivo/directorio origen
- `-destino` (Obligatorio): Ubicación donde copiar

**Efectos en las Estructuras**:

- **Creación**: Agrega nueva entrada en directorio destino
- **Referencia**: Apunta al mismo inodo del archivo original
- **Incremento**: Aumenta el contador de referencias del inodo
- **Journaling**: Registra la operación de copia

**Diferencia con MOVE**: COPY mantiene el archivo en ambas ubicaciones, MOVE lo reubica.

**Ejemplo**:

```bash
copy -path=/home/user/importante.txt -destino=/home/backup/
copy -path=/home/proyecto -destino=/home/respaldos/proyecto_copia
```

### 22. RENAME

**Propósito**: Cambia el nombre de un archivo o directorio.

**Parámetros**:

- `-path` (Obligatorio): Ruta del archivo/directorio a renombrar
- `-name` (Obligatorio): Nuevo nombre

**Efectos en las Estructuras**:

- **Modificación**: Actualiza la entrada en el bloque de directorio padre
- **Preservación**: Mantiene el mismo inodo y datos
- **Journaling**: Registra la operación de renombrado

**Ejemplo**:

```bash
rename -path=/home/user/archivo_viejo.txt -name=archivo_nuevo.txt
rename -path=/home/carpeta_temporal -name=carpeta_final
```

### 23. CHMOD

**Propósito**: Cambia los permisos de un archivo o directorio.

**Parámetros**:

- `-path` (Obligatorio): Ruta del archivo/directorio
- `-ugo` (Obligatorio): Permisos en formato octal (ej: 755, 644)
- `-r` (Opcional): Aplicar recursivamente a subdirectorios

**Efectos en las Estructuras**:

- **Modificación**: Actualiza el campo `i_perm` del inodo
- **Recursión**: Si `-r` está presente, aplica a todos los elementos contenidos
- **Journaling**: Registra el cambio de permisos

**Formato de Permisos UGO**:

- **Usuario (U)**: Propietario del archivo
- **Grupo (G)**: Miembros del grupo propietario
- **Otros (O)**: Resto de usuarios

**Ejemplo**:

```bash
chmod -ugo=755 -path=/home/user/script.sh
chmod -ugo=644 -path=/home/user/documento.txt -r
```

### 24. CHOWN

**Propósito**: Cambia el propietario de un archivo o directorio.

**Parámetros**:

- `-path` (Obligatorio): Ruta del archivo/directorio
- `-user` (Obligatorio): Nuevo usuario propietario
- `-r` (Opcional): Aplicar recursivamente

**Efectos en las Estructuras**:

- **Modificación**: Actualiza los campos `i_uid` e `i_gid` del inodo
- **Validación**: Verifica que el usuario exista en users.txt
- **Recursión**: Si `-r` está presente, aplica a contenido del directorio

**Restricciones**:

- Solo root puede cambiar propietarios
- El usuario destino debe existir en el sistema

**Ejemplo**:

```bash
chown -user=usuario1 -path=/home/shared/archivo.txt
chown -user=admin -path=/home/proyecto -r
```

### 25. FIND

**Propósito**: Busca archivos y directorios que coincidan con un patrón de nombre.

**Parámetros**:

- `-path` (Obligatorio): Directorio donde iniciar la búsqueda
- `-name` (Obligatorio): Patrón de nombre a buscar (acepta wildcards)

**Efectos en las Estructuras**:

- **Lectura**: Recorre recursivamente la estructura de directorios
- **Búsqueda**: Compara nombres con el patrón especificado
- **Recolección**: Genera lista de coincidencias encontradas

**Patrones Soportados**:

- Búsqueda exacta: `archivo.txt`
- Comodines: `*.txt`, `temp*`, `*backup*`

**Ejemplo**:

```bash
find -path=/home -name="*.txt"
find -path=/home/user -name="backup*"
find -path=/ -name="config.json"
```

### 26. REMOVE

**Propósito**: Elimina archivos o directorios del sistema de archivos.

**Parámetros**:

- `-path` (Obligatorio): Ruta del elemento a eliminar

**Efectos en las Estructuras**:

- **Liberación**: Marca el inodo como libre en bitmap de inodos
- **Liberación**: Libera todos los bloques usados en bitmap de bloques
- **Eliminación**: Remueve entrada del directorio padre
- **Recursión**: Para directorios, elimina todo el contenido
- **Journaling**: Registra la operación de eliminación

**Validaciones**:

- Verifica permisos de escritura
- Para directorios, valida que el usuario tenga permisos sobre el contenido

**Ejemplo**:

```bash
remove -path=/home/user/archivo_temporal.txt
remove -path=/home/user/carpeta_vieja
```

### 27. UNMOUNT

**Propósito**: Desmonta una partición previamente montada del sistema.

**Parámetros**:

- `-id` (Obligatorio): ID de la partición montada a desmontar

**Efectos en las Estructuras**:

- **Memoria**: Remueve la partición del registro de montajes en RAM
- **Estado**: Actualiza el estado de montaje de la partición
- **Limpieza**: Libera recursos asociados a la partición

**Validaciones**:

- Verifica que la partición esté efectivamente montada
- No permite desmontar si hay sesión activa en la partición

**Ejemplo**:

```bash
unmount -id=341A
```

### 28. PDISK

**Propósito**: Muestra información detallada de las particiones de un disco.

**Parámetros**:

- `-path` (Obligatorio): Ruta del archivo de disco (.mia)

**Efectos en las Estructuras**:

- **Lectura**: Accede al MBR del disco
- **Análisis**: Lee la tabla de particiones
- **Formato**: Presenta información en formato tabular

**Información Mostrada**:

- Número de partición
- Tipo (Primaria, Extendida, Lógica)
- Estado (Activa/Inactiva)
- Tamaño en bytes
- Nombre de la partición

**Ejemplo**:

```bash
pdisk -path=/home/user/disco1.mia
```

## Comandos de Journaling y Recuperación

### 29. RECOVERY

**Propósito**: Recupera el sistema de archivos EXT3 utilizando el journal de transacciones.

**Parámetros**:

- `-id` (Obligatorio): ID de la partición a recuperar

**Efectos en las Estructuras**:

- **Análisis**: Lee el journal de transacciones
- **Recuperación**: Aplica las transacciones pendientes
- **Restauración**: Restablece la consistencia del sistema de archivos
- **Validación**: Verifica la integridad post-recuperación

**Casos de Uso**:

- Recuperación después de una pérdida de datos simulada
- Restauración de transacciones incompletas
- Reparación de inconsistencias en el sistema de archivos

**Ejemplo**:

```bash
recovery -id=341A
```

### 30. LOSS

**Propósito**: Simula la pérdida del sistema de archivos para pruebas de recuperación.

**Parámetros**:

- `-id` (Obligatorio): ID de la partición donde simular la pérdida

**Efectos en las Estructuras**:

- **Simulación**: Corrompe estructuras del sistema de archivos
- **Testing**: Permite probar funcionalidades de recuperación
- **Journaling**: Mantiene el journal intacto para posterior recuperación

**Uso Educativo**: Comando diseñado para demostrar capacidades de recuperación.

**Ejemplo**:

```bash
loss -id=341A
```

### 31. JOURNALING

**Propósito**: Muestra información detallada del journal de transacciones EXT3.

**Parámetros**:

- `-id` (Obligatorio): ID de la partición cuyo journal consultar

**Información Mostrada**:

- Lista de operaciones registradas
- Timestamps de cada transacción
- Estado de las transacciones (PENDING, COMMITTED, ABORTED)
- Detalles de archivos y operaciones afectadas

**Efectos en las Estructuras**:

- **Solo Lectura**: No modifica estructuras, solo consulta
- **Análisis**: Presenta el contenido del journal de forma legible

**Ejemplo**:

```bash
journaling -id=341A
```

## Sistema de Journaling EXT3

### Funcionalidad del Journal

El sistema implementa journaling EXT3 que registra automáticamente todas las operaciones del sistema de archivos:

**Operaciones Registradas**:

- Creación de archivos y directorios (`MKFILE`, `MKDIR`)
- Modificación de contenido (`EDIT`)
- Operaciones de movimiento y copia (`MOVE`, `COPY`)
- Cambios de permisos y propietarios (`CHMOD`, `CHOWN`)
- Eliminación de elementos (`REMOVE`)
- Renombrado (`RENAME`)

**Estructura de Entrada de Journal**:

```go
type JournalEntry struct {
    PartitionID string    // ID de la partición
    Operation   string    // Tipo de operación
    Path        string    // Ruta del archivo/directorio afectado
    Content     string    // Detalles adicionales de la operación
    Timestamp   time.Time // Momento de la operación
    Status      string    // Estado: PENDING, COMMITTED, ABORTED
}
```

**Ventajas del Journaling**:

- **Consistencia**: Garantiza integridad ante fallos del sistema
- **Recuperación**: Permite restaurar el estado anterior a errores
- **Auditoría**: Mantiene historial completo de operaciones
- **Debugging**: Facilita el diagnóstico de problemas

## Frontend - Gestor de Archivos Integrado

### Nuevas Funcionalidades del Frontend

El frontend ha sido extendido con un **Gestor de Archivos** visual que permite:

**Navegación Visual**:

- Selección de discos disponibles
- Exploración de particiones montadas
- Navegación por directorios
- Visualización de contenido de archivos

**Componentes Nuevos**:

```
frontend/src/
├── views/
│   ├── Home.vue          # Vista principal con navegación
│   └── SystemFiles.vue   # Gestor de archivos
├── components/layout/
│   ├── Disk_layout.vue      # Visualización de discos
│   ├── Partition_layout.vue # Visualización de particiones
│   ├── Directory_layout.vue # Explorador de directorios
│   ├── File_layout.vue      # Visualizador de archivos
│   └── Journal_layout.vue   # Interfaz del journal
└── store/
    ├── disks.js         # Estado de discos y particiones
    └── files.js         # Estado del sistema de archivos
```

**Funcionalidades del Gestor**:

- **Navegación jerárquica**: Disco → Partición → Directorio → Archivo
- **Visualización de contenido**: Lectura de archivos de texto
- **Journal integrado**: Acceso visual al historial de operaciones
- **Transiciones animadas**: Experiencia de usuario fluida

---

## Notas Importantes

### Sistema de Permisos

- **Usuario Root**: Tiene permisos 777 sobre todos los archivos y carpetas
- **Permisos UGO**: User, Group, Other en formato octal
- **Permisos por Defecto**: 664 para archivos y carpetas nuevos

### Limitaciones y Restricciones

- Solo particiones primarias pueden ser montadas para el sistema de archivos
- Tamaño de bloque fijo: 64 bytes para contenido útil
- Máximo 4 entradas por bloque de carpeta
- Sistema case-sensitive para usuarios y grupos

### Estructura del Archivo users.txt

```
1, G, root
1, U, root, root, 123
2, G, nombre_grupo
2, U, nombre_grupo, nombre_usuario, contraseña
```

### Comandos que Requieren Sesión Activa

Todos los comandos excepto MKFS y LOGIN requieren una sesión activa iniciada con LOGIN.

### Integración Frontend-Backend

**Flujo de Datos para Gestor de Archivos**:

```
Frontend: Selección de disco/partición
    ↓
Backend: Validación de partición montada y sesión activa
    ↓
Backend: Lectura de estructura de directorios desde inodos
    ↓
Frontend: Renderizado visual de archivos y carpetas
    ↓
Usuario: Navegación y selección de elementos
    ↓
Backend: Lectura de contenido de archivos específicos
    ↓
Frontend: Visualización de contenido en interfaz
```

**Comunicación Journal**:

- El journal se actualiza automáticamente en operaciones de escritura
- El frontend puede consultar el journal en tiempo real
- Las entradas se muestran con formato temporal legible
- Filtrado por tipo de operación y rango de fechas

**AWS**:

![alt text](image-8.png)
