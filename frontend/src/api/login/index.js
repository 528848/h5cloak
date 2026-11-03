import request from '@/utils/request';
import { APP_PORTAL } from '@portal';

/**
 * 登录。携带当前构建所属门户，后端校验账号归属。
 */
export async function login(data) {
  const res = await request.post('/login', { ...data, portal: APP_PORTAL });
  if (res.data.code === 200) {
    return { ...(res.data.data || {}), msg: res.data.msg };
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 获取验证码
 */
export async function getCaptcha() {
  const res = await request.get('/captchaImage');
  if (res.data.code === 200) {
    return res.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 退出登录
 */
export async function logout() {
  const res = await request.post('/logout');
  if (res.data.code === 200) {
    return res.data.msg;
  }
  return Promise.reject(new Error(res.data.msg));
}
