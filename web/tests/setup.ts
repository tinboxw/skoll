import { vi } from "vitest";

class ResizeObserverMock {
	observe(): void {}
	unobserve(): void {}
	disconnect(): void {}
}

vi.stubGlobal("ResizeObserver", ResizeObserverMock);
vi.stubGlobal("matchMedia", vi.fn().mockImplementation((query: string) => ({
	matches: false,
	media: query,
	onchange: null,
	addEventListener: vi.fn(),
	removeEventListener: vi.fn(),
	addListener: vi.fn(),
	removeListener: vi.fn(),
	dispatchEvent: vi.fn()
})));
