/// <reference types="vite/client" />

declare module "*.vue" {
  import type { DefineComponent } from "vue";
  const component: DefineComponent<{}, {}, any>;
  export default component;
}

// Declaraciones para archivos JavaScript del store
declare module "../store/user" {
  export function useUserStore(): any;
}

declare module "../store/disks" {
  export function usedisksStore(): any;
}

declare module "../store/files" {
  export function usefilesStore(): any;
}

// También declaramos con la extensión .js por si acaso
declare module "../store/user.js" {
  export function useUserStore(): any;
}

declare module "../store/disks.js" {
  export function usedisksStore(): any;
}

declare module "../store/files.js" {
  export function usefilesStore(): any;
}
