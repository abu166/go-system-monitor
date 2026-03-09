import { describe, expect, it } from 'vitest';
import { getReconnectDelayMs } from './api';

describe('getReconnectDelayMs', () => {
  it('grows exponentially and caps at max delay', () => {
    const base = 1000;
    const max = 10000;

    expect(getReconnectDelayMs(0, base, max)).toBe(1000);
    expect(getReconnectDelayMs(1, base, max)).toBe(2000);
    expect(getReconnectDelayMs(2, base, max)).toBe(4000);
    expect(getReconnectDelayMs(3, base, max)).toBe(8000);
    expect(getReconnectDelayMs(4, base, max)).toBe(10000);
    expect(getReconnectDelayMs(8, base, max)).toBe(10000);
  });

  it('defends against invalid values', () => {
    expect(getReconnectDelayMs(-1, 0, 0)).toBeGreaterThan(0);
    expect(getReconnectDelayMs(1, -100, -200)).toBeGreaterThan(0);
  });
});
