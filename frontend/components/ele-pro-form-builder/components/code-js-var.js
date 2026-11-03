import { codeStringPrefix } from '../../ele-pro-form/components/render-core';

/**
 * 标识 js 变量名
 */
export class JsVar {
  /** 生成代码时显示的变量名称 */
  name;
  /** 生成 ElForm 代码时添加的声明代码 */
  code;
  /** 生成 ProForm 代码时添加的声明代码 */
  proCode;
  /** 生成 ElForm 代码时添加的导入代码 */
  imports;
  /** 生成 ProForm 代码时添加的导入代码 */
  proImports;
  /** 原始数据类型 */
  dataType;
  /** 转 JSON 时标识是 JsVar 对象 */
  __isJsVar = true;

  constructor(props) {
    this.name = props.name;
    this.code = props.code;
    this.proCode = props.proCode;
    this.imports = props.imports;
    this.proImports = props.proImports;
    this.dataType = props.dataType;
  }
}

/**
 * 移除字符串前缀
 * @param content 字符串
 * @param prefix 一个或多个前缀
 */
export function removePrefixStr(content, prefix) {
  if (typeof prefix === 'string') {
    if (content.startsWith(prefix)) {
      return content.slice(prefix.length);
    }
    return content;
  }
  let result = content;
  prefix.forEach((temp) => {
    result = removePrefixStr(result, temp);
  });
  return result;
}

/**
 * JSON 数据值转换 JsVar 对象
 * @param jv JSON 解析后的值
 */
export function transformJsVar(jv) {
  if (jv != null) {
    if (typeof jv === 'string') {
      if (jv.startsWith(codeStringPrefix)) {
        const codeName = removePrefixStr(jv, [codeStringPrefix]);
        return new JsVar({ name: codeName });
      }
    }
    if (typeof jv === 'object') {
      if (jv instanceof JsVar) {
        return jv;
      }
      if (jv.__isJsVar === true && jv.name != null) {
        return new JsVar(jv);
      }
    }
  }
}
