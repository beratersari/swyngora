import { configureStore } from '@reduxjs/toolkit';
import { setupListeners } from '@reduxjs/toolkit/query';
import { baseApi } from './baseApi';
import './endpoints/healthApi';
import './endpoints/marketApi';
import './endpoints/watchlistApi'
import './endpoints/alertsApi';
import './endpoints/aiApi';
import './endpoints/portfolioApi';
import './endpoints/accountApi';
import './endpoints/exportApi';
import './endpoints/recurringApi';
import './endpoints/scannerApi';
import './endpoints/swingApi';

export const store = configureStore({
  reducer: {
    [baseApi.reducerPath]: baseApi.reducer,
  },
  middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(baseApi.middleware),
});

// Enables refetchOnFocus / refetchOnReconnect for RTK Query
setupListeners(store.dispatch);

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
