import { listDepts } from '@/api/system/dept';
import { pageUsers } from '@/api/system/user';

/**
 * 部门树id字段名
 */
export const deptIdField = 'deptId';

/**
 * 部门树名称字段名
 */
export const deptNameField = 'deptName';

/**
 * 部门树id字段名
 */
export const deptParentIdField = 'parentId';

/**
 * 部门树查询接口
 */
export const listDeptApi = listDepts;

/**
 * 用户id字段名
 */
export const userIdField = 'userId';

/**
 * 用户名字段名
 */
export const userNameField = 'nickName';

/**
 * 用户头像字段名
 */
export const avatarField = 'avatar';

/**
 * 用户分页查询接口
 */
export const pageUserApi = async (params) => {
  // 修改搜索条件字段名
  params.nickName = params.nickname;
  params.phonenumber = params.phone;
  params.nickname = void 0;
  params.phone = void 0;
  const result = await pageUsers(params);
  return result;
};

/**
 * 用户表格列
 */
export const userTableColumns = [
  {
    prop: 'userName',
    label: 'labelUsername',
    align: 'center',
    minWidth: 100
  },
  {
    prop: 'nickName',
    label: 'labelNickname',
    align: 'center',
    minWidth: 110
  },
  {
    prop: 'dept.deptName',
    label: 'labelOrganizationName',
    align: 'center',
    minWidth: 100
  },
  {
    prop: 'phonenumber',
    label: 'labelPhone',
    align: 'center',
    minWidth: 110
  },
  {
    prop: 'status',
    label: 'labelStatus',
    width: 80,
    align: 'center',
    slot: 'status'
  }
];

/**
 * 用户状态字典
 */
export const statusDictCode = 'sys_normal_disable';
