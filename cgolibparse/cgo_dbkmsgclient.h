//
// Created by sfere on 2025/12/26.
//

#ifndef ARM_DBKMSGCLIENT_CGO_H
#define ARM_DBKMSGCLIENT_CGO_H
#include <stdint.h>
#include <stdlib.h>

/**
 *Created by sunlianhao on 2025/12/25.
 *V0.1  每次调用业务接口，内部均新建 zmq 连接,结束后销毁
 *多线程调用时，必须由外部确保 传入的 client_id 唯一
 *否则可能消息混乱，可能返回-1，可能返回错误数据
*/


#ifdef __cplusplus
extern "C"{
#endif

    // JSON结果结构体
    typedef struct JsonResult{
        char* data;      // JSON字符串  通过 delete 释放
        int length;      // JSON长度
        int error_code;  // 错误码 0 为正确  -1 操作数据库错误 -2 内存分配错误 -3 参数错误 -4请求超时 -5网络发送错误
    } JsonResult;

    JsonResult InitJsonResult(){
        JsonResult json;
        json.data = NULL;
        json.length = 0;
        json.error_code = 0;
        return json;
    }

    typedef struct KV{
        uint64_t oid;
        char* field;        //最大64
        char* value;        //最大64
    } KV;
    // typedef void* DbHandle;
    /**
     * 调用库前初始化，内置zmq客户端
     * @param clientID 不同调用者确保唯一
     * @param timeoutMs 毫秒，请求通用超时时间  5*1000
     * @return 0成功， -1失败
     */
    void DBCGOInit();

    void SetTimeOutMs(int timeoutMs);

    /**
     * 释放无用的json
     * @param json
     */
    void DestroyJsonResult(JsonResult* json);

    /**
     * 获取指定oid下，指定字段的值
     * @param oid
     * @param field
     * @param json 出参
     * @return <0失败
     */
    int GetFieldValue(const char* clientID,uint64_t oid, const char* field, JsonResult* json);
    /**
     * 获取批量oid下，指定字段的值
     * @param oids
     * @param count oid的数量
     * @param field
     * @param json 出参
     * @return <0失败
     */
    int GetFieldValues(const char* clientID,uint64_t* oids, int count, const char* field, JsonResult* json);

    int GetFieldsValues(const char *clientID, uint64_t *oids, int count, const char **field,int fCound, JsonResult *json);
    /**
     * 获得指定oid下，所有数据
     * @param oid
     * @param json 出参
     * @return <0失败
     */
    int GetObjectByID(const char* clientID,uint64_t oid, JsonResult* json);
    /**
     * 获取指定表下,指定rtu的数量
     * @param nTable 表号
     * @param nRtu rtu号
     * @param json 出参
     * @return <0失败
     */
    int GetRtuObjCount(const char* clientID,uint8_t nTable, uint16_t nRtu, JsonResult* json);
    /**
     * 获取指定表下,指定rtu的所有数据
     * @param nTable
     * @param nRtu
     * @param json 出参
     * @return <0失败
     */
    int GetAllRtuObjects(const char* clientID,uint8_t nTable, uint16_t nRtu, JsonResult* json);
    /**
     * 获取指定表下，指定dev的数量
     * @param nTable
     * @param nDev
     * @param json 出参
     * @return <0失败
     */
    int GetDevObjCount(const char* clientID,uint8_t nTable, uint16_t nRtu,uint16_t nDev, JsonResult* json);
    /**
     * 获取指定表下，指定dev的所有数据
     * @param nTable
     * @param nDev
     * @param json 出参
     * @return <0失败
     */
    int GetAllDevObjects(const char* clientID,uint8_t nTable,uint16_t nRtu, uint16_t nDev, JsonResult* json);

    /**
     * 获取指定表下，所有数量
     * @param nTable
     * @param json 出参
     * @return <0失败
     */
    int GetObjCount(const char* clientID,uint8_t nTable, JsonResult* json);
    /**
     * 获取指定表下，所有数据
     * @param nTable
     * @param json 出参
     * @return <0失败
     */
    int GetAllObjects(const char* clientID,uint8_t nTable, JsonResult* json);
    /**
     * 更新指定oid下，指定字段的值
     * @param oid
     * @param strField
     * @param strValue
     * @param json 出参
     * @return <0失败
     */
    int UpdateFieldValue(const char* clientID,uint64_t oid, const char* strField, const char* strValue, JsonResult* json);
    /**
     * 更新批量oid下，指定字段的值
     * @param kv
     * @param count
     * @param json 出参
     * @return <0失败
     */
    int UpdateFieldValues(const char* clientID,KV* kv,int count, JsonResult* json);
    /**
     * 更新指定oid下，所有数据 （批量）req中携带数组
     * @param req 入参
     * @param json 出参
     * @return <0失败
     */
    int UpdateObjectByID(const char* clientID,JsonResult* req, JsonResult* json);
    /**
     * 插入指定oid数据 （批量）req中携带数组
     * @param req 入参 同update
     * @param json 出参
     * @return <0失败
     */
    int InsertObjects(const char* clientID,JsonResult* req, JsonResult* json);
    /**
     * 删除指定oid
     * @param oid
     * @param json 出参
     * @return <0失败
     */
    int DeleteObjectByID(const char* clientID,uint64_t oid,JsonResult* json);
    /**
     * 删除批量oid
     * @param oid
     * @param count
     * @param json 出参
     * @return <0失败
     */
    int DeleteObjectsByID(const char* clientID,uint64_t *oid,int count,JsonResult* json);
    /**
     * 删除指定表下，所有数据
     * @param nTable
     * @param json
     * @return <0失败
     */
    int DeleteAllObjects(const char* clientID,uint8_t nTable,JsonResult* json);
    /**
     * 删除指定表下，指定RTU数据
     * @param nTable
     * @param nRtu
     * @param json
     * @return <0失败
     */
    int DeleteObjectsByRtu(const char* clientID,uint8_t nTable, uint16_t nRtu,JsonResult* json);
    /**
     * 删除指定表下，指定Dev数据
     * @param nTable
     * @param nDev
     * @param json
     * @return <0失败
     */
    int DeleteObjectsByDev(const char* clientID,uint8_t nTable, uint16_t nRtu, uint16_t nDev,JsonResult* json);

    /**
     * 实时库加载xml
     * @param clientID
     * @param json
     * @return 异步接口，返回值不代表执行结果
     */
    int LoadFile(const char* clientID,JsonResult* json);

#ifdef __cplusplus
}
#endif

#endif // ARM_DBKMSGCLIENT_CGO_H