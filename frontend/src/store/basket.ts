import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { BasketLine, Product, ProductSize } from '@/types';

interface BasketState {
  lines: BasketLine[];
  freeSampleSku: string | null;
  giftWrap: boolean;
  giftMessage: string;
  discountCode: string | null;
  discountAmount: number;
  /** Set when the user just added something — drives the "Added" toast. */
  lastAdded: { name: string; at: number } | null;

  add: (product: Pick<Product, 'id' | 'slug' | 'name' | 'images' | 'tags'>, size: ProductSize, qty?: number) => void;
  setQty: (sku: string, qty: number) => void;
  remove: (sku: string) => void;
  clear: () => void;
  setFreeSample: (sku: string | null) => void;
  setGiftWrap: (on: boolean, message?: string) => void;
  applyDiscount: (code: string, amount: number) => void;
  clearDiscount: () => void;
}

export const selectCount = (s: BasketState) => s.lines.reduce((n, l) => n + l.qty, 0);
export const selectSubtotal = (s: BasketState) => s.lines.reduce((n, l) => n + l.unitPrice * l.qty, 0);

export const useBasketStore = create<BasketState>()(
  persist(
    (set) => ({
      lines: [],
      freeSampleSku: null,
      giftWrap: false,
      giftMessage: '',
      discountCode: null,
      discountAmount: 0,
      lastAdded: null,

      add: (product, size, qty = 1) =>
        set((s) => {
          const existing = s.lines.find((l) => l.sku === size.sku);
          const isSet = product.tags.includes('set') || product.tags.includes('gift');
          const lines = existing
            ? s.lines.map((l) => (l.sku === size.sku ? { ...l, qty: Math.min(10, l.qty + qty) } : l))
            : [
                ...s.lines,
                {
                  sku: size.sku,
                  productId: product.id,
                  slug: product.slug,
                  name: product.name,
                  variantLabel: isSet ? size.label : `Eau de Parfum · ${size.label}`,
                  image: product.images[0],
                  unitPrice: size.price,
                  qty: Math.min(10, qty),
                },
              ];
          return { lines, lastAdded: { name: product.name, at: Date.now() } };
        }),
      setQty: (sku, qty) =>
        set((s) => ({
          lines: qty <= 0 ? s.lines.filter((l) => l.sku !== sku) : s.lines.map((l) => (l.sku === sku ? { ...l, qty: Math.min(10, qty) } : l)),
        })),
      remove: (sku) => set((s) => ({ lines: s.lines.filter((l) => l.sku !== sku) })),
      clear: () => set({ lines: [], freeSampleSku: null, giftWrap: false, giftMessage: '', discountCode: null, discountAmount: 0 }),
      setFreeSample: (sku) => set({ freeSampleSku: sku }),
      setGiftWrap: (on, message) => set((s) => ({ giftWrap: on, giftMessage: message ?? s.giftMessage })),
      applyDiscount: (code, amount) => set({ discountCode: code, discountAmount: amount }),
      clearDiscount: () => set({ discountCode: null, discountAmount: 0 }),
    }),
    {
      name: 'co.basket',
      version: 1,
      partialize: (s) => ({
        lines: s.lines,
        freeSampleSku: s.freeSampleSku,
        giftWrap: s.giftWrap,
        giftMessage: s.giftMessage,
        discountCode: s.discountCode,
        discountAmount: s.discountAmount,
      }),
    },
  ),
);
