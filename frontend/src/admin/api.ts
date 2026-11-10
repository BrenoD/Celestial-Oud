import { useMockApi } from '@/config';
import { mockAdminApi } from './api.mock';
import { realAdminApi } from './api.real';
import type { AdminApi } from './types';

export const adminApi: AdminApi = useMockApi ? mockAdminApi : realAdminApi;
