import request from '@/utils/request';

/**
 * 获取当前登录用户的个人信息
 * @param toRoute 路由守卫中要进入的路由
 */
export async function getUserInfo(toRoute) {
  const res = await request.get('/getInfo', { toRoute });
  if (res.data.code === 200) {
    return res.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 获取当前登录用户的菜单
 * @param toRoute 路由守卫中要进入的路由
 */
export async function getUserMenu(toRoute) {
  const res = await request.get('/getRouters', { toRoute });
  if (res.data.code === 200 && res.data.data) {
    return res.data.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 修改当前登录用户的密码
 */
export async function updatePassword(data) {
  const res = await request.put('/system/user/profile/updatePwd', data);
  if (res.data.code === 200) {
    return res.data.msg;
  }
  return Promise.reject(new Error(res.data.msg));
}
