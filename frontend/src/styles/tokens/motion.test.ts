import { describe, expect, it } from 'vitest';
import { motion, motionDuration, motionEase } from './motion';

describe('motion tokens', () => {
  it('keeps desk motion under a second', () => {
    expect(Number.parseInt(motionDuration.fast, 10)).toBeLessThan(200);
    expect(Number.parseInt(motionDuration.base, 10)).toBeLessThan(300);
    expect(Number.parseInt(motionDuration.slow, 10)).toBeLessThan(800);
  });

  it('exposes standard easing on the theme bundle', () => {
    expect(motion.ease.standard).toBe(motionEase.standard);
    expect(motion.duration.base).toBe(motionDuration.base);
  });
});
