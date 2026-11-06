import { formatPence, formatPenceShort } from './money';

describe('money', () => {
  it('formats pence as GBP', () => {
    expect(formatPence(4500)).toBe('£45.00');
    expect(formatPence(499)).toBe('£4.99');
  });
  it('drops .00 in the short form', () => {
    expect(formatPenceShort(4500)).toBe('£45');
    expect(formatPenceShort(4550)).toBe('£45.50');
  });
});
