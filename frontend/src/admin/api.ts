import { mockAdminApi } from './api.mock';
import { realAdminApi } from './api.real';
import type { AdminApi } from './types';

// Build-time constant so production bundles drop the mock.
export const adminApi: AdminApi = import.meta.env.VITE_API_URL ? realAdminApi : mockAdminApi;
