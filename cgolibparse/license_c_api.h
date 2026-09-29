#ifndef LICENSE_C_API_H
#define LICENSE_C_API_H

/*
 * license_manager 的 C 语言 ABI 封装，供 Go (cgo) / C / 其它语言调用。
 *
 * 内存约定：所有返回 char* 的函数，其内存由本库用 malloc 分配，
 * 调用方用完必须调用 lm_free() 释放；返回 NULL 表示失败或结果为空。
 * LMLicenseInfo 内的每个 char* 字段同样需要单独 lm_free()。
 */

#ifdef __cplusplus
extern "C" {
#endif

/* 许可证信息（C 版本，expire_date 为 "YYYY-MM-DD" 字符串） */
typedef struct {
    int   is_valid;        /* 1 有效，0 无效 */
    int   remaining_days;  /* 剩余天数，无效时为 0 */
    char* serial_number;   /* 机器序列号，可能为 NULL */
    char* license_key;     /* 传入的注册码，可能为 NULL */
    char* expire_date;     /* 过期日期 "YYYY-MM-DD"，仅在 is_valid 时非 NULL */
} LMLicenseInfo;

/* 采集本机硬件信息，生成唯一序列号，形如 XXXX-XXXX-XXXX-XXXX */
char* lm_generate_machine_serial(void);

/* 根据序列号与过期日期（"YYYY-MM-DD"）生成注册码 */
char* lm_generate_license_key(const char* serial_number, const char* expire_date);

/* 验证注册码，返回许可证信息 */
LMLicenseInfo lm_validate_license(const char* license_key);

/* 保存注册码到文件；file_path 传 NULL 时使用默认 "license.dat"。
 * 返回 1 成功，0 失败。 */
int lm_save_license_to_file(const char* license_key, const char* file_path);

/* 从文件加载注册码；file_path 传 NULL 时使用默认 "license.dat"。
 * 加载并验证成功返回注册码，否则返回 NULL。 */
char* lm_load_license_from_file(const char* file_path);

/* 检查是否过期：1 已过期/无效，0 有效未过期 */
int lm_is_expired(void);

/* 获取剩余天数，无效许可证返回 0 */
int lm_get_remaining_days(void);

/* 启动/停止后台定期检查（默认间隔 3600 秒）。
 * 注意：后台线程检测到过期时会直接 exit(1) 结束进程。 */
void lm_start_license_check(int check_interval_seconds);
void lm_stop_license_check(void);
int  lm_is_running(void);

/* 释放本库返回的字符串；str 为 NULL 时安全 */
void lm_free(char* str);

#ifdef __cplusplus
}
#endif

#endif /* LICENSE_C_API_H */
