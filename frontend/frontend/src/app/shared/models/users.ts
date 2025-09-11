export interface RegisterUserModel {
  name?: string;
  email?: string;
  phone?: string;
  role?: string;
}

export interface LoginUserModel {
  email?: string;
}

export interface UserModel {
  id: string;
  email: string;
  name: string;
  phone: string;
  role: string;
}
export interface LoginUserResponseModel {
  status: number;
  access_token: string;
  refresh_token: string;
  user: UserModel;
}
