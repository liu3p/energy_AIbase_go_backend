#ifndef __CGO_LIBPARSE_H__
#define __CGO_LIBPARSE_H__

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/*
cJsonStr := C.getStrategyLibInfo(path)
if cJsonStr == nil {
    fmt.Println("获取 JSON 数据失败")
    return
}
defer C.JSON_Free(cJsonStr) 

*/
const char* getStrategyLibInfo(char* libpath);
void JSON_Free(char* json_str);

const char* getPluginLibInfo();

int subscribeFeChannelStatus();
int subscribeFeChannelLogs(uint32_t fechannelID, uint32_t pluginID);
// int notifyFeChannelReload();
int notifyFeServerReload(uint32_t channel_ids[],int number);
int recvFeChannelMsg(char** outputJson);


#ifdef __cplusplus
}
#endif

#endif // __CGO_LIBPARSE_H__

