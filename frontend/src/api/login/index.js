import request from '@/utils/request';

/**
 * 登录
 */
export async function login(data) {
  const res = await request.post('/login', data);
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
