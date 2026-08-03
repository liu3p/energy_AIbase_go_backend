#pragma once

#ifdef __cplusplus
extern "C" {
#endif

/**************************************************
 * 函数名: eq_validate
 * 功能: 验证公式是否合法(自动分解变量名,此处规则是单个字母加数字的组合)
 * 参数: expr:待验证的公式
 *       err_msg:错误信息
 * 返回值:
 *  0  合法
 *  <0  不合法
 **************************************************/
int eq_validate(const char* expr, const char** err_msg);

/**************************************************
 * 函数名: eq_validate_with_vars
 * 功能: 使用调用方传入的变量名，验证公式是否合法
 * 参数: expr    : 待验证公式
 *   var_names  : 变量名数组
 *   var_count  : 变量个数
 *   err_msg    : 错误信息（需要调用 eq_free_string 释放）
 * 返回值:
 *   0  合法
 *  <0  不合法
 **************************************************/
int eq_validate_with_vars(const char* expr,
                            const char** var_names,
                            int var_count,
                            const char** err_msg);

/**************************************************
 * 函数名: eq_eval
 * 功能: 计算公式
 * 参数: expr:待计算的公式
 *       var_names:变量名数组
 *       var_values:变量值数组
 *       var_count:变量个数
 *       result:计算结果
 *       err_msg:错误信息
 * 返回值:成功返回0,失败返回值小于0
 * **************************************************/
int eq_eval(const char* expr,
            const char** var_names,
            const double* var_values,
            int var_count,
            double* result,
            const char** err_msg);

/**************************************************
 * 函数名: eq_free_string
 * 功能: 释放字符串
 * 参数: s:待释放的字符串
 * 返回值: 无
 **************************************************/
void eq_free_string(const char* s);

#ifdef __cplusplus
}
#endif