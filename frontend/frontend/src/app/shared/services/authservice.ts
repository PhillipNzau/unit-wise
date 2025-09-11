import { Inject, Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router } from '@angular/router';
import { map } from 'rxjs/operators';

// Import the injection token and interface

import { API_CONFIG, ApiConfig } from '../../api.config';
import {
  LoginUserModel,
  LoginUserResponseModel,
  RegisterUserModel,
} from '../models/users';

@Injectable({
  providedIn: 'root',
})
export class Authservice {
  private loggedIn = false;
  private apiConfig = inject(API_CONFIG);

  router = inject(Router);
  http = inject(HttpClient);

  // login user
  loginUser(userData: LoginUserModel) {
    const url = `${this.apiConfig.baseUrl}${this.apiConfig.endpoints.loginUser}`;
    console.log('====================================');
    console.log(url);
    console.log('====================================');
    return this.http.post<LoginUserResponseModel>(url, userData).pipe(
      map((res) => {
        if (res.status === 200) {
          return res;
        }
        return res;
      })
    );
  }

  // signup user
  registerUser(userData: RegisterUserModel) {
    const url = `${this.apiConfig.baseUrl}${this.apiConfig.endpoints.registerUser}`;
    return this.http.post<LoginUserResponseModel>(url, userData).pipe(
      map((res) => {
        if (res.status === 200) {
          return res;
        }
        return res;
      })
    );
  }

  refreshToken(refresh: string) {
    const url = `${this.apiConfig.baseUrl}${this.apiConfig.endpoints.refreshToken}`;
    const body = {
      refresh_token: refresh,
    };
    return this.http.post<any>(url, body).pipe(
      map((res) => {
        if (res.status === 200) {
          localStorage.setItem('uWfTk', JSON.stringify(res.access_token));
          localStorage.setItem('uWfRTk', JSON.stringify(res.refresh_token));
          localStorage.setItem('cnLguWf', 'true');
          this.loggedIn = !!localStorage.getItem('cnLguWf');

          return res;
        }
        return res;
      })
    );
  }

  verifyOtp(otpBody: any) {
    const url = `${this.apiConfig.baseUrl}${this.apiConfig.endpoints.verifyOtp}`;
    return this.http.post<any>(url, otpBody).pipe(
      map((res) => {
        if (res.status === 200) {
          localStorage.setItem('uWfUsr', JSON.stringify(res.user));
          localStorage.setItem('uWfTk', JSON.stringify(res.access_token));
          localStorage.setItem('uWfRTk', JSON.stringify(res.refresh_token));
          localStorage.setItem('cnLguWf', 'true');
          this.loggedIn = !!localStorage.getItem('cnLguWf');

          return res;
        }
        return res;
      })
    );
  }

  // Returns true when user is logged in and email is verified
  get isLoggedIn() {
    this.loggedIn = !!localStorage.getItem('cnLguWf');

    if (!this.loggedIn) {
      // You can return the promise from the router's navigate call directly
      return this.router.navigate(['/auth']);
    }
    return this.loggedIn;
  }
}
