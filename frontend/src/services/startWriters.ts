import type { Ref } from "vue";

function startTypewriter(
  newTitle: string,
  typingInterval: Ref<number>,
  typewriterText: Ref<string>
): void {
  clearInterval(typingInterval.value);
  let current = typewriterText.value;
  let phase = "deleting";
  let i = current.length;
  let j = 0;

  typingInterval.value = setInterval(() => {
    if (phase === "deleting") {
      if (i > 0) {
        typewriterText.value = current.slice(0, i - 1);
        i--;
      } else {
        phase = "writing";
      }
    } else if (phase === "writing") {
      if (j < newTitle.length) {
        typewriterText.value += newTitle[j];
        j++;
      } else {
        clearInterval(typingInterval.value);
      }
    }
  }, 60); // velocidad de animación
}

function startPathTypewriter(
  newPathArr: string[],
  oldPathArr: string[] = [],
  typingPathInterval: Ref<number>,
  typewriterPath: Ref<string>
) {
  clearInterval(typingPathInterval.value);

  // Convierte arrays a string
  const newPathStr = newPathArr.join("");
  const oldPathStr = oldPathArr.join("");

  // Detecta el punto de diferencia
  let commonLength = 0;
  while (
    commonLength < oldPathArr.length &&
    commonLength < newPathArr.length &&
    oldPathArr[commonLength] === newPathArr[commonLength]
  ) {
    commonLength++;
  }

  // Si se eliminó, animar borrado
  if (oldPathArr.length > newPathArr.length) {
    let i = oldPathStr.length;
    typingPathInterval.value = setInterval(() => {
      if (i > newPathStr.length) {
        typewriterPath.value = oldPathStr.slice(0, i - 1);
        i--;
      } else {
        clearInterval(typingPathInterval.value);
      }
    }, 60);
    return;
  }

  // Si se agregó, animar solo el fragmento nuevo
  const baseStr = newPathArr.slice(0, commonLength).join("");
  const fragment = newPathArr.slice(commonLength).join("");
  let j = 0;
  typewriterPath.value = baseStr;
  typingPathInterval.value = setInterval(() => {
    if (j < fragment.length) {
      typewriterPath.value = baseStr + fragment.slice(0, j + 1);
      j++;
    } else {
      clearInterval(typingPathInterval.value);
    }
  }, 60);
}

export { startTypewriter, startPathTypewriter };
