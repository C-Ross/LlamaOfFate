import "@testing-library/jest-dom/vitest"

Object.defineProperty(globalThis, "localStorage", {
  configurable: true,
  value: window.localStorage,
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
