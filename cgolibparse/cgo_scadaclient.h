#pragma once

#include <stdint.h>

#if defined(_WIN32) || defined(_WIN64)
    #ifdef SCADA_CLIENT_EXPORTS
        #define SCADA_CLIENT_API __declspec(dllexport)
    #else
        #define SCADA_CLIENT_API __declspec(dllimport)
    #endif
#else
    #define SCADA_CLIENT_API __attribute__((visibility("default")))
#endif

#ifdef __cplusplus
extern "C" {
#endif

/* ================== 客户端句柄 ================== */
typedef void* scada_client_t;

/* ================== 遥控遥调阶段控制 ================== */
typedef enum {
    SCADA_CONTROL_SELECT = 0,
    SCADA_CONTROL_CANCEL,
    SCADA_CONTROL_EXECUTE,
    SCADA_CONTROL_PAUSE
} scada_control_type_t;

/* ================== 遥控动作类型 ================== */
typedef enum {
    SCADA_ACTION_ON = 0,
    SCADA_ACTION_OFF,
    SCADA_ACTION_ERROR,
    SCADA_ACTION_STOP
} scada_action_type_t;

/* ================== 人工置数类型选择 ================== */
typedef enum {
    SCADA_SET_ANALOG = 0,
    SCADA_SET_DIGITAL,
    SCADA_SET_PULSE
} scada_set_type_t;

/* ================== 生命周期 ================== */

SCADA_CLIENT_API scada_client_t scada_client_create(const char* client_name);
SCADA_CLIENT_API void scada_client_destroy(scada_client_t client);

/**************************************************
 * 函数名:scada_send_yt_cmd
 * 功能:发送遥调命令
 * 参数:client:客户端句柄
 *      poid:遥调对象id
 *      cmd:遥控遥调阶段控制
 *      value:遥调值
 *      is_fast_yt:是否快速遥调,1为快速遥调,0为普通遥调
 *返回值:成功返回0,失败返回-1 
***************************************************/
SCADA_CLIENT_API int scada_send_yt_cmd(scada_client_t client,
                                        uint64_t poid,
                                        scada_control_type_t cmd,
                                        float value,
                                        int is_fast_yt);
/**************************************************
 * 函数名:scada_send_yk_cmd
 * 功能:发送遥控命令
 * 参数:client:客户端句柄
 *      poid:遥控对象id
 *      cmd:遥控遥调阶段控制
 *      value:遥控动作
 *返回值:成功返回0,失败返回-1 
***************************************************/
SCADA_CLIENT_API int scada_send_yk_cmd(scada_client_t client,
                                        uint64_t poid,
                                        scada_control_type_t cmd,
                                        scada_action_type_t value);
/**************************************************
 * 函数名:scada_send_manual_set_cmd
 * 功能:发送人工置数命令
 * 参数:client:客户端句柄
 *      poid:人工置数对象id
 *      set_flag:人工置数标志,0为取消人工置数,1为人工置数
 *      value:人工置数值
 *      set_type:人工置数类型
 * 返回值:成功返回0,失败返回-1
**************************************************/
SCADA_CLIENT_API int scada_send_manual_set_cmd(scada_client_t client,
                                                uint64_t poid,
                                                uint8_t set_flag,
                                                double value,
                                                scada_set_type_t set_type);

#ifdef __cplusplus
}
#endif
