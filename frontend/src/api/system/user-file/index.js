//import request from '@/utils/request';

/**
 * 分页查询用户文件
 */
export async function pageUserFiles(params) {
  console.log('pageUserFiles params:', params);
  const res = {
    data: {
      code: 0,
      message: '操作成功',
      data: {
        list: [
          {
            id: 159,
            userId: 40,
            name: 'avatar.jpg',
            isDirectory: 0,
            parentId: 0,
            path: '20220722/19b623b787514f8eb7d1e8edf2e9ff15.jpg',
            length: 12805,
            contentType: 'image/jpeg',
            deleted: 0,
            tenantId: 4,
            createTime: '2022-07-22 11:28:16',
            updateTime: '2022-07-22 16:08:31',
            url: 'https://v2.eleadmin.com/api/file/20220722/19b623b787514f8eb7d1e8edf2e9ff15.jpg',
            thumbnail:
              'https://v2.eleadmin.com/api/file/thumbnail/20220722/19b623b787514f8eb7d1e8edf2e9ff15.jpg',
            downloadUrl:
              'https://v2.eleadmin.com/api/file/download/20220722/19b623b787514f8eb7d1e8edf2e9ff15.jpg'
          },
          {
            id: 160,
            userId: 40,
            name: 'logo.svg',
            isDirectory: 0,
            parentId: 0,
            path: '20220722/e77995c100de46409cce84e848a63f7d.svg',
            length: 1605,
            contentType: 'image/svg+xml',
            deleted: 0,
            tenantId: 4,
            createTime: '2022-07-22 11:28:21',
            updateTime: '2022-07-22 16:08:32',
            url: 'https://v2.eleadmin.com/api/file/20220722/e77995c100de46409cce84e848a63f7d.svg',
            thumbnail:
              'https://v2.eleadmin.com/api/file/thumbnail/20220722/e77995c100de46409cce84e848a63f7d.svg',
            downloadUrl:
              'https://v2.eleadmin.com/api/file/download/20220722/e77995c100de46409cce84e848a63f7d.svg'
          },
          {
            id: 161,
            userId: 40,
            name: 'favicon.ico',
            isDirectory: 0,
            parentId: 0,
            path: '20220722/acb31ddb5ce44530933c48741c3bbfe2.ico',
            length: 4286,
            contentType: 'image/vnd.microsoft.icon',
            deleted: 0,
            tenantId: 4,
            createTime: '2022-07-22 11:29:11',
            updateTime: '2022-07-22 16:08:32',
            url: 'https://v2.eleadmin.com/api/file/20220722/acb31ddb5ce44530933c48741c3bbfe2.ico',
            thumbnail:
              'https://v2.eleadmin.com/api/file/thumbnail/20220722/acb31ddb5ce44530933c48741c3bbfe2.ico',
            downloadUrl:
              'https://v2.eleadmin.com/api/file/download/20220722/acb31ddb5ce44530933c48741c3bbfe2.ico'
          },
          {
            id: 162,
            userId: 40,
            name: '用户导入模板.xlsx',
            isDirectory: 0,
            parentId: 0,
            path: '20220722/b4fa501bf6084f2e8bed971e68462ce3.xlsx',
            length: 10221,
            contentType:
              'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
            deleted: 0,
            tenantId: 4,
            createTime: '2022-07-22 11:29:24',
            updateTime: '2022-07-22 16:08:35',
            url: 'https://v2.eleadmin.com/api/file/20220722/b4fa501bf6084f2e8bed971e68462ce3.xlsx',
            thumbnail: null,
            downloadUrl:
              'https://v2.eleadmin.com/api/file/download/20220722/b4fa501bf6084f2e8bed971e68462ce3.xlsx'
          },
          {
            id: 163,
            userId: 40,
            name: 'classes.json',
            isDirectory: 0,
            parentId: 0,
            path: '20220722/8d59c7d48453444d9ee84bd2b4c27a12.json',
            length: 6789,
            contentType: 'application/json',
            deleted: 0,
            tenantId: 4,
            createTime: '2022-07-22 11:29:28',
            updateTime: '2022-07-22 16:08:34',
            url: 'https://v2.eleadmin.com/api/file/20220722/8d59c7d48453444d9ee84bd2b4c27a12.json',
            thumbnail: null,
            downloadUrl:
              'https://v2.eleadmin.com/api/file/download/20220722/8d59c7d48453444d9ee84bd2b4c27a12.json'
          },
          {
            id: 166,
            userId: 40,
            name: 'pro202012301.png',
            isDirectory: 0,
            parentId: 157,
            path: '20220722/4cbac74d9c5c45858854b24ee5db8e87.png',
            length: 59257,
            contentType: 'image/png',
            deleted: 0,
            tenantId: 4,
            createTime: '2022-07-22 11:34:39',
            updateTime: '2022-07-22 16:08:37',
            url: 'https://v2.eleadmin.com/api/file/20220722/4cbac74d9c5c45858854b24ee5db8e87.png',
            thumbnail:
              'https://v2.eleadmin.com/api/file/thumbnail/20220722/4cbac74d9c5c45858854b24ee5db8e87.png',
            downloadUrl:
              'https://v2.eleadmin.com/api/file/download/20220722/4cbac74d9c5c45858854b24ee5db8e87.png'
          },
          {
            id: 167,
            userId: 40,
            name: 'pro202012302.png',
            isDirectory: 0,
            parentId: 157,
            path: '20220722/24ba2b0906f64394ae0fb5e8f813cb30.png',
            length: 64832,
            contentType: 'image/png',
            deleted: 0,
            tenantId: 4,
            createTime: '2022-07-22 11:34:43',
            updateTime: '2022-07-22 16:08:38',
            url: 'https://v2.eleadmin.com/api/file/20220722/24ba2b0906f64394ae0fb5e8f813cb30.png',
            thumbnail:
              'https://v2.eleadmin.com/api/file/thumbnail/20220722/24ba2b0906f64394ae0fb5e8f813cb30.png',
            downloadUrl:
              'https://v2.eleadmin.com/api/file/download/20220722/24ba2b0906f64394ae0fb5e8f813cb30.png'
          },
          {
            id: 168,
            userId: 40,
            name: 'pro202012303.png',
            isDirectory: 0,
            parentId: 157,
            path: '20220722/9d269d1fe7294ea4b5ca6f11c6ac8a95.png',
            length: 74518,
            contentType: 'image/png',
            deleted: 0,
            tenantId: 4,
            createTime: '2022-07-22 11:34:47',
            updateTime: '2022-07-22 16:08:39',
            url: 'https://v2.eleadmin.com/api/file/20220722/9d269d1fe7294ea4b5ca6f11c6ac8a95.png',
            thumbnail:
              'https://v2.eleadmin.com/api/file/thumbnail/20220722/9d269d1fe7294ea4b5ca6f11c6ac8a95.png',
            downloadUrl:
              'https://v2.eleadmin.com/api/file/download/20220722/9d269d1fe7294ea4b5ca6f11c6ac8a95.png'
          },
          {
            id: 169,
            userId: 40,
            name: 'eleadmin20201020172822.png',
            isDirectory: 0,
            parentId: 157,
            path: '20220722/45c686ab54264822963bc40716222591.png',
            length: 103558,
            contentType: 'image/png',
            deleted: 0,
            tenantId: 4,
            createTime: '2022-07-22 13:02:52',
            updateTime: '2022-07-22 16:08:39',
            url: 'https://v2.eleadmin.com/api/file/20220722/45c686ab54264822963bc40716222591.png',
            thumbnail:
              'https://v2.eleadmin.com/api/file/thumbnail/20220722/45c686ab54264822963bc40716222591.png',
            downloadUrl:
              'https://v2.eleadmin.com/api/file/download/20220722/45c686ab54264822963bc40716222591.png'
          },
          {
            id: 170,
            userId: 40,
            name: 'RZ8FQmZfHkcffMlTBCJllBFjEhEsObVo.jpg',
            isDirectory: 0,
            parentId: 164,
            path: '20220722/099d2f77dfad41c7af776ad81af7aab5.jpg',
            length: 38206,
            contentType: 'image/jpeg',
            deleted: 0,
            tenantId: 4,
            createTime: '2022-07-22 13:34:41',
            updateTime: '2022-07-22 16:08:40',
            url: 'https://v2.eleadmin.com/api/file/20220722/099d2f77dfad41c7af776ad81af7aab5.jpg',
            thumbnail:
              'https://v2.eleadmin.com/api/file/thumbnail/20220722/099d2f77dfad41c7af776ad81af7aab5.jpg',
            downloadUrl:
              'https://v2.eleadmin.com/api/file/download/20220722/099d2f77dfad41c7af776ad81af7aab5.jpg'
          },
          {
            id: 171,
            userId: 40,
            name: 'WLXm7gp1EbLDtvVQgkeQeyq5OtDm00Jd.jpg',
            isDirectory: 0,
            parentId: 164,
            path: '20220722/faf7fec118824f309fb66961866d9712.jpg',
            length: 60365,
            contentType: 'image/jpeg',
            deleted: 0,
            tenantId: 4,
            createTime: '2022-07-22 13:34:49',
            updateTime: '2022-07-22 16:08:41',
            url: 'https://v2.eleadmin.com/api/file/20220722/faf7fec118824f309fb66961866d9712.jpg',
            thumbnail:
              'https://v2.eleadmin.com/api/file/thumbnail/20220722/faf7fec118824f309fb66961866d9712.jpg',
            downloadUrl:
              'https://v2.eleadmin.com/api/file/download/20220722/faf7fec118824f309fb66961866d9712.jpg'
          },
          {
            id: 172,
            userId: 40,
            name: '4Z0QR2L0J1XStxBh99jVJ8qLfsGsOgjU.jpg',
            isDirectory: 0,
            parentId: 164,
            path: '20220722/d845f1a4e91143dc81ec9fa96ce6f074.jpg',
            length: 50118,
            contentType: 'image/jpeg',
            deleted: 0,
            tenantId: 4,
            createTime: '2022-07-22 13:34:54',
            updateTime: '2022-07-22 16:08:42',
            url: 'https://v2.eleadmin.com/api/file/20220722/d845f1a4e91143dc81ec9fa96ce6f074.jpg',
            thumbnail:
              'https://v2.eleadmin.com/api/file/thumbnail/20220722/d845f1a4e91143dc81ec9fa96ce6f074.jpg',
            downloadUrl:
              'https://v2.eleadmin.com/api/file/download/20220722/d845f1a4e91143dc81ec9fa96ce6f074.jpg'
          },
          {
            id: 173,
            userId: 40,
            name: 'ttkIjNPlVDuv4lUTvRX8GIlM2QqSe0jg.jpg',
            isDirectory: 0,
            parentId: 164,
            path: '20220722/4fc98decedcb4029bc954369c5e8c294.jpg',
            length: 29948,
            contentType: 'image/jpeg',
            deleted: 0,
            tenantId: 4,
            createTime: '2022-07-22 13:35:03',
            updateTime: '2022-07-22 16:08:42',
            url: 'https://v2.eleadmin.com/api/file/20220722/4fc98decedcb4029bc954369c5e8c294.jpg',
            thumbnail:
              'https://v2.eleadmin.com/api/file/thumbnail/20220722/4fc98decedcb4029bc954369c5e8c294.jpg',
            downloadUrl:
              'https://v2.eleadmin.com/api/file/download/20220722/4fc98decedcb4029bc954369c5e8c294.jpg'
          },
          {
            id: 174,
            userId: 40,
            name: 'fAenQ8nvRjL7x0i0jEfuDBZHvJfHf3v6.jpg',
            isDirectory: 0,
            parentId: 164,
            path: '20220722/9787d2f5b37d4813a48b72e950d00395.jpg',
            length: 48228,
            contentType: 'image/jpeg',
            deleted: 0,
            tenantId: 4,
            createTime: '2022-07-22 13:35:08',
            updateTime: '2022-07-22 16:08:43',
            url: 'https://v2.eleadmin.com/api/file/20220722/9787d2f5b37d4813a48b72e950d00395.jpg',
            thumbnail:
              'https://v2.eleadmin.com/api/file/thumbnail/20220722/9787d2f5b37d4813a48b72e950d00395.jpg',
            downloadUrl:
              'https://v2.eleadmin.com/api/file/download/20220722/9787d2f5b37d4813a48b72e950d00395.jpg'
          },
          {
            id: 175,
            userId: 40,
            name: 'LrCTN2j94lo9N7wEql7cBr1Ux4rHMvmZ.jpg',
            isDirectory: 0,
            parentId: 164,
            path: '20220722/9dacd9d48efa483ab204ae8ad0b60864.jpg',
            length: 37320,
            contentType: 'image/jpeg',
            deleted: 0,
            tenantId: 4,
            createTime: '2022-07-22 13:35:12',
            updateTime: '2022-07-22 16:08:43',
            url: 'https://v2.eleadmin.com/api/file/20220722/9dacd9d48efa483ab204ae8ad0b60864.jpg',
            thumbnail:
              'https://v2.eleadmin.com/api/file/thumbnail/20220722/9dacd9d48efa483ab204ae8ad0b60864.jpg',
            downloadUrl:
              'https://v2.eleadmin.com/api/file/download/20220722/9dacd9d48efa483ab204ae8ad0b60864.jpg'
          },
          {
            id: 176,
            userId: 40,
            name: 'yeKvhT20lMU0f1T3Y743UlGEOLLnZSnp.jpg',
            isDirectory: 0,
            parentId: 164,
            path: '20220722/f74da7fd19524bc0a4e75e6159d041ac.jpg',
            length: 29040,
            contentType: 'image/jpeg',
            deleted: 0,
            tenantId: 4,
            createTime: '2022-07-22 13:35:17',
            updateTime: '2022-07-22 16:08:44',
            url: 'https://v2.eleadmin.com/api/file/20220722/f74da7fd19524bc0a4e75e6159d041ac.jpg',
            thumbnail:
              'https://v2.eleadmin.com/api/file/thumbnail/20220722/f74da7fd19524bc0a4e75e6159d041ac.jpg',
            downloadUrl:
              'https://v2.eleadmin.com/api/file/download/20220722/f74da7fd19524bc0a4e75e6159d041ac.jpg'
          },
          {
            id: 177,
            userId: 40,
            name: 'CyrCNmTJfv7D6GFAg39bjT3eRkkRm5dI.jpg',
            isDirectory: 0,
            parentId: 164,
            path: '20220722/7539f0f90c4749cbbb56b111e2b8d6af.jpg',
            length: 39560,
            contentType: 'image/jpeg',
            deleted: 0,
            tenantId: 4,
            createTime: '2022-07-22 13:35:26',
            updateTime: '2022-07-22 16:08:45',
            url: 'https://v2.eleadmin.com/api/file/20220722/7539f0f90c4749cbbb56b111e2b8d6af.jpg',
            thumbnail:
              'https://v2.eleadmin.com/api/file/thumbnail/20220722/7539f0f90c4749cbbb56b111e2b8d6af.jpg',
            downloadUrl:
              'https://v2.eleadmin.com/api/file/download/20220722/7539f0f90c4749cbbb56b111e2b8d6af.jpg'
          },
          {
            id: 178,
            userId: 40,
            name: 'faa0202700ee455b90fe77d8bef98bc0.jpg',
            isDirectory: 0,
            parentId: 165,
            path: '20220722/5079a6bc43b64bf2b86b87275dfaf7a2.jpg',
            length: 10250,
            contentType: 'image/jpeg',
            deleted: 0,
            tenantId: 4,
            createTime: '2022-07-22 13:36:21',
            updateTime: '2022-07-22 16:08:45',
            url: 'https://v2.eleadmin.com/api/file/20220722/5079a6bc43b64bf2b86b87275dfaf7a2.jpg',
            thumbnail:
              'https://v2.eleadmin.com/api/file/thumbnail/20220722/5079a6bc43b64bf2b86b87275dfaf7a2.jpg',
            downloadUrl:
              'https://v2.eleadmin.com/api/file/download/20220722/5079a6bc43b64bf2b86b87275dfaf7a2.jpg'
          }
        ],
        count: 18
      }
    }
  };
  if (res.data.code === 0) {
    return res.data.data;
  }
  return Promise.reject(new Error(res.data.message));
}

/**
 * 查询用户文件列表
 */
export async function listUserFiles(params) {
  console.log('listUserFiles params:', params);
  const res = {
    data: {
      code: 0,
      message: '操作成功',
      data: [
        {
          id: 157,
          userId: 40,
          name: '我的图片',
          isDirectory: 1,
          parentId: 0,
          path: null,
          length: null,
          contentType: null,
          deleted: 0,
          tenantId: 4,
          createTime: '2022-07-22 11:28:02',
          updateTime: '2022-07-22 16:08:28',
          url: null,
          thumbnail: null,
          downloadUrl: null
        },
        {
          id: 158,
          userId: 40,
          name: '我的文件',
          isDirectory: 1,
          parentId: 0,
          path: null,
          length: null,
          contentType: null,
          deleted: 0,
          tenantId: 4,
          createTime: '2022-07-22 11:28:09',
          updateTime: '2022-07-22 16:08:30',
          url: null,
          thumbnail: null,
          downloadUrl: null
        },
        {
          id: 164,
          userId: 40,
          name: '壁纸',
          isDirectory: 1,
          parentId: 157,
          path: null,
          length: null,
          contentType: null,
          deleted: 0,
          tenantId: 4,
          createTime: '2022-07-22 11:30:58',
          updateTime: '2022-07-22 16:08:36',
          url: null,
          thumbnail: null,
          downloadUrl: null
        },
        {
          id: 165,
          userId: 40,
          name: '头像',
          isDirectory: 1,
          parentId: 157,
          path: null,
          length: null,
          contentType: null,
          deleted: 0,
          tenantId: 4,
          createTime: '2022-07-22 11:31:02',
          updateTime: '2022-07-22 16:08:37',
          url: null,
          thumbnail: null,
          downloadUrl: null
        }
      ]
    }
  };
  if (res.data.code === 0 && res.data.data) {
    return res.data.data;
  }
  return Promise.reject(new Error(res.data.message));
}

/**
 * 添加用户文件
 */
export async function addUserFile(data) {
  console.log('addUserFile data:', data);
  const res = { data: { code: 404, message: '接口不存在' } };
  if (res.data.code === 0) {
    return res.data.message;
  }
  return Promise.reject(new Error(res.data.message));
}

/**
 * 修改用户文件
 */
export async function updateUserFile(data) {
  console.log('updateUserFile data:', data);
  const res = { data: { code: 404, message: '接口不存在' } };
  if (res.data.code === 0) {
    return res.data.message;
  }
  return Promise.reject(new Error(res.data.message));
}

/**
 * 删除用户文件
 */
export async function removeUserFile(id) {
  console.log('removeUserFile id:', id);
  const res = { data: { code: 404, message: '接口不存在' } };
  if (res.data.code === 0) {
    return res.data.message;
  }
  return Promise.reject(new Error(res.data.message));
}

/**
 * 批量删除用户文件
 */
export async function removeUserFiles(data) {
  console.log('removeUserFiles data:', data);
  const res = { data: { code: 404, message: '接口不存在' } };
  if (res.data.code === 0) {
    return res.data.message;
  }
  return Promise.reject(new Error(res.data.message));
}
