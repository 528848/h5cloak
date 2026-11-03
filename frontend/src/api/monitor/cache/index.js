import request from '@/utils/request';

/**
 * 查询缓存信息
 */
export async function getCache() {
  const res = await request.get('/monitor/cache');
  if (res.data.code === 200) {
    // 演示接口无数据添加模拟数据
    if (res.data.data.info == null || res.data.data.commandStats == null) {
      return JSON.parse(
        '{"commandStats":[{"name":"exists","value":"1"},{"name":"set","value":"18"},{"name":"setex","value":"4"},{"name":"del","value":"3"},{"name":"info","value":"5"},{"name":"ping","value":"1"},{"name":"dbsize","value":"2"},{"name":"get","value":"25"}],"info":{"aof_enabled":"0","redis_mode":"standalone","rdb_last_bgsave_status":"ok","tcp_port":"6379","used_memory_human":"720.15K","used_cpu_user_children":"0.000000","connected_clients":"1","redis_version":"5.0.14.1","maxmemory_human":"0B","uptime_in_days":"0","instantaneous_input_kbps":"0.00"},"dbSize":17}'
      );
    }
    return res.data.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 查询缓存列表
 */
export async function getCacheNames() {
  const res = await request.get('/monitor/cache/getNames');
  if (res.data.code === 200) {
    return res.data.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 查询缓存键名
 */
export async function getCacheKeys(name) {
  const res = await request.get('/monitor/cache/getKeys/' + name);
  if (res.data.code === 200) {
    return res.data.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 查询缓存数据
 */
export async function getCacheValue(name, key) {
  const res = await request.get('/monitor/cache/getValue/' + name + '/' + key);
  if (res.data.code === 200) {
    return res.data.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 清空缓存数据
 */
export async function clearCacheName(name) {
  const res = await request.delete('/monitor/cache/clearCacheName/' + name);
  if (res.data.code === 200) {
    return res.data.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 清空缓存数据
 */
export async function clearCacheKey(key) {
  const res = await request.delete('/monitor/cache/clearCacheKey/' + key);
  if (res.data.code === 200) {
    return res.data.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 清空全部缓存数据
 */
export async function clearCacheAll() {
  const res = await request.delete('/monitor/cache/clearCacheAll');
  if (res.data.code === 200) {
    return res.data.msg;
  }
  return Promise.reject(new Error(res.data.msg));
}
