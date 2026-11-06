/**
 * TanStack Query hooks. Query keys are centralised so cache invalidation is
 * predictable; staleTime keeps the catalogue warm while a visitor browses so
 * repeated navigation does not hit the API again.
 */
import { keepPreviousData, useMutation, useQuery } from '@tanstack/react-query';
import { api } from './index';
import type { CreateOrderRequest, ProductQuery } from '@/types';

export const keys = {
  collections: ['collections'] as const,
  collection: (slug: string) => ['collection', slug] as const,
  products: (q: ProductQuery) => ['products', q] as const,
  product: (slug: string) => ['product', slug] as const,
  related: (slug: string) => ['related', slug] as const,
  reviews: (slug: string) => ['reviews', slug] as const,
  delivery: ['delivery-methods'] as const,
  orders: ['orders'] as const,
  order: (id: string) => ['order', id] as const,
  faqs: ['faqs'] as const,
};

const FIVE_MIN = 5 * 60_000;

export const useCollections = () =>
  useQuery({ queryKey: keys.collections, queryFn: api.listCollections, staleTime: FIVE_MIN });

export const useCollection = (slug: string) =>
  useQuery({ queryKey: keys.collection(slug), queryFn: () => api.getCollection(slug), staleTime: FIVE_MIN });

export const useProducts = (q: ProductQuery) =>
  useQuery({
    queryKey: keys.products(q),
    queryFn: ({ signal }) => api.listProducts(q, signal),
    staleTime: 60_000,
    placeholderData: keepPreviousData,
  });

export const useProduct = (slug: string) =>
  useQuery({ queryKey: keys.product(slug), queryFn: () => api.getProduct(slug), staleTime: FIVE_MIN, enabled: !!slug });

export const useRelated = (slug: string) =>
  useQuery({ queryKey: keys.related(slug), queryFn: () => api.listRelated(slug), staleTime: FIVE_MIN, enabled: !!slug });

export const useReviews = (slug: string) =>
  useQuery({ queryKey: keys.reviews(slug), queryFn: () => api.listReviews(slug), staleTime: FIVE_MIN, enabled: !!slug });

export const useDeliveryMethods = () =>
  useQuery({ queryKey: keys.delivery, queryFn: api.listDeliveryMethods, staleTime: FIVE_MIN });

export const useOrders = (enabled: boolean) =>
  useQuery({ queryKey: keys.orders, queryFn: api.listOrders, enabled, staleTime: 30_000 });

export const useOrder = (id: string, email?: string) =>
  useQuery({ queryKey: keys.order(id), queryFn: () => api.getOrder(id, email), enabled: !!id, staleTime: FIVE_MIN });

export const useFaqs = () => useQuery({ queryKey: keys.faqs, queryFn: api.listFaqs, staleTime: Infinity });

export const useCreateOrder = () =>
  useMutation({
    mutationFn: ({ req, idempotencyKey }: { req: CreateOrderRequest; idempotencyKey: string }) =>
      api.createOrder(req, idempotencyKey),
  });

export const useNewsletter = () => useMutation({ mutationFn: (email: string) => api.subscribeNewsletter(email) });
export const useContact = () => useMutation({ mutationFn: api.contact });
export const useSignIn = () =>
  useMutation({ mutationFn: ({ email, password }: { email: string; password: string }) => api.signIn(email, password) });
export const useRegister = () => useMutation({ mutationFn: api.register });
export const useValidateDiscount = () =>
  useMutation({ mutationFn: ({ code, subtotal }: { code: string; subtotal: number }) => api.validateDiscount(code, subtotal) });
export const usePostcodeLookup = () => useMutation({ mutationFn: (postcode: string) => api.lookupPostcode(postcode) });
