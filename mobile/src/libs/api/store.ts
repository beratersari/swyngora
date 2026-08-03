import { configureStore } from '@reduxjs/toolkit';
import { setupListeners } from '@reduxjs/toolkit/query';
import { baseApi } from './baseApi';
import './endpoints/healthApi';
import './endpoints/marketApi';
import './endpoints/watchlistApi';
import './endpoints/pumpApi';
import './endpoints/aiApi';

export const store = configureStore({
  reducer: {
    [baseApi.reducerPath]: baseApi.reducer,
  },
  middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(baseApi.middleware),
});

// Note: browser focus listeners from setupListeners are incomplete on RN.
// Mobile uses AppState pollingInterval in ViewModels (refetchOnFocus: false).
setupListeners(store.dispatch);

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
