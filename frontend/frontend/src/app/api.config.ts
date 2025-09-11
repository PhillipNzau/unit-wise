import { InjectionToken } from '@angular/core';

// Define the shape of your API configuration object
// Define the shape of your API configuration object
export interface ApiConfig {
  baseUrl: string;
  endpoints: {
    loginUser: string;
    registerUser: string;
    refreshToken: string;
    verifyOtp: string;
    // Add other endpoints as needed
  };
}

// Create the injection token
export const API_CONFIG = new InjectionToken<ApiConfig>('api.config');

// Define the actual configuration value
export const apiConfigValue: ApiConfig = {
  baseUrl: 'https://api.your-app.com/v1',
  endpoints: {
    loginUser: '/auth/login', // New endpoint
    registerUser: '/auth/register', // New endpoint
    refreshToken: '/auth/refresh', // New endpoint
    verifyOtp: '/auth/verify-otp', // New endpoint
  },
};
