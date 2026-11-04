import "@testing-library/jest-dom/vitest"

const storage = new Map<string, string>()

Object.defineProperty(window, "localStorage", {
  configurable: true,
  value: {
    get length() {
      return storage.size
    },
    clear: () => storage.clear(),
    getItem: (key: string) => storage.get(String(key)) ?? null,
    key: (index: number) => [...storage.keys()][index] ?? null,
    removeItem: (key: string) => {
      storage.delete(String(key))
    },
    setItem: (key: string, value: string) => {
      storage.set(String(key), String(value))
    },
  } satisfies Storage,
})

// Mock window.matchMedia — required by sonner (toast library) in jsdom.
Object.defineProperty(window, "matchMedia", {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }),
})
