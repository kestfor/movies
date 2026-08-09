import { describe, expect, it } from 'vitest';
import { shouldConfirmTitleLoaded } from './titleOpenFeedback';

describe('shouldConfirmTitleLoaded', () => {
  it('confirms completion after an actual load', () => {
    expect(shouldConfirmTitleLoaded(false)).toBe(true);
  });

  it('does not add a second haptic for cached data', () => {
    expect(shouldConfirmTitleLoaded(true)).toBe(false);
  });
});
