function formatearFecha(fechaStr: string): string {
  // Reemplaza el espacio por 'T' para que Date lo reconozca correctamente
  const fecha = new Date(fechaStr.replace(" ", "T"));
  const opciones: Intl.DateTimeFormatOptions = {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  };
  return fecha.toLocaleString("es-ES", opciones);
}

export { formatearFecha };
