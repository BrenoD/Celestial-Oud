import { selectCount, selectSubtotal, useBasketStore } from './basket';
import { products } from '@/data/catalogue';

const royal = products[0];

beforeEach(() => {
  useBasketStore.getState().clear();
});

describe('basket store', () => {
  it('adds lines and merges duplicates by sku', () => {
    const { add } = useBasketStore.getState();
    add(royal, royal.sizes[1]);
    add(royal, royal.sizes[1], 2);
    add(royal, royal.sizes[0]);
    const s = useBasketStore.getState();
    expect(s.lines).toHaveLength(2);
    expect(selectCount(s)).toBe(4);
    expect(selectSubtotal(s)).toBe(royal.sizes[1].price * 3 + royal.sizes[0].price);
  });

  it('caps quantity at 10 and removes at 0', () => {
    const { add, setQty } = useBasketStore.getState();
    add(royal, royal.sizes[1], 20);
    expect(useBasketStore.getState().lines[0].qty).toBe(10);
    setQty(royal.sizes[1].sku, 0);
    expect(useBasketStore.getState().lines).toHaveLength(0);
  });

  it('clear resets extras', () => {
    const s = useBasketStore.getState();
    s.add(royal, royal.sizes[1]);
    s.setGiftWrap(true, 'Happy birthday');
    s.applyDiscount('WELCOME10', 550);
    s.clear();
    const after = useBasketStore.getState();
    expect(after.giftWrap).toBe(false);
    expect(after.discountCode).toBeNull();
    expect(after.lines).toHaveLength(0);
  });
});
