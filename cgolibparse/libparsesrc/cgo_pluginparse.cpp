#include "cgo_libparse.h"

#include "utility/product_line.h"
#include "utility/t_link.h"
#include <unistd.h> 
#include <signal.h>
#include <Poco/JSON/Parser.h>
#include <Poco/JSON/Object.h>
#include "stdio.h"

#include "fe_model/fe_model_manage.h"
#include "fe_model/fe_model_common.h"


UI4 m_ui4_slot_id = 0x10002;
t_smart_ptr<i_mail_slot> m_sp_mail_slot = c_mail_slot_factory::create_mail_slot(
			1,m_ui4_slot_id,4096,4096*1024,true);

class PluginLifecycleManager {
public:
    PluginLifecycleManager() {
        // make sure only call once
        printf("load_all_model_and_plugin...");
        FES::load_all_model_and_plugin();
    }
    ~PluginLifecycleManager() {
        printf("unload_all_model_and_plugin...");
        FES::unload_all_model_and_plugin();
    }
};

const char* getPluginLibInfo() {
    static PluginLifecycleManager plugin_lifecycle_manager;

    Poco::JSON::Array::Ptr pluginsInfo = new Poco::JSON::Array();

    TFC_UTILITY::t_set<UI4> fac_ids;
    c_work_shop_factory::enum_factory_ids(fac_ids);
    printf("factory size:%d\n",fac_ids.size());

    for(int index=0;index<fac_ids.size();index++)
    {
        UI4 id = fac_ids.get_data(index);
        TFC_UTILITY::t_link<TFC_UTILITY::c_parameter_description *> all_param;
        TFC_UTILITY::i_worker_factory * p_wf = c_work_shop_factory::find_factory(id);
        if(p_wf == NULL) 
        {
            printf("find factory id:%d NULL!\n",id);
            continue;
        }
        p_wf->all_parameter_description(all_param);

        printf("factory id:%ld,name:%s\n",id,p_wf->get_name().c_str());

        Poco::JSON::Object::Ptr pluginInfo = new Poco::JSON::Object;
        Poco::JSON::Object::Ptr paramsInfo = new Poco::JSON::Object;
        pluginInfo->set("id", id);
        pluginInfo->set("name", p_wf->get_name());

        TFC_UTILITY::c_handle h = all_param.first_node();
        for(I4 i = 0; i < all_param.size() && HANDLE_VALID(h); i++ , h = all_param.get_next_handle(h))
        {
            TFC_UTILITY::c_parameter_description * p_param_desp = NULL;
            make_sure(all_param.get_data(h,p_param_desp));

            if(p_param_desp == NULL ) {
                return; 
            }

            Poco::JSON::Object::Ptr paramInfo = new Poco::JSON::Object;
            paramInfo->set("default", p_param_desp->m_str_default_value);
            paramInfo->set("valuelist", p_param_desp->m_str_valuelist);

            printf("\t%s:%s,%s\n", p_param_desp->m_str_name.c_str(),p_param_desp->m_str_default_value.c_str(), p_param_desp->m_str_valuelist.c_str());
            paramsInfo->set(p_param_desp->m_str_name, paramInfo);
        }
        pluginInfo->set("params", paramsInfo);
        pluginsInfo->add(pluginInfo);
    }

    std::ostringstream oss;
    pluginsInfo->stringify(oss);
    std::string json_str = oss.str();

    char* ret_json = (char*)malloc(json_str.length() + 1);
    if (ret_json == nullptr) {
        printf("getPluginLibInfo malloc json_str error\n");
        return nullptr;
    }
    strcpy(ret_json, json_str.c_str());

    // 外部接口释放该内存
    return ret_json;
}

int subscribeFeChannelStatus() {
    setvbuf(stdout, NULL, _IONBF, 0); // _IONBF=无缓冲
    setvbuf(stderr, NULL, _IONBF, 0);
    printf(">>>subscribeFeChannelStatus in\n");
    fflush(stdout);
    if (!m_sp_mail_slot.valid()) {
        printf("subscribeFeChannelStatus m_sp_mail_slot is invalid\n");
        return -2;
    }
    c_mail msg;
    msg.m_ui4_qualify = 0;
	msg.m_ui4_type = FES::MT_SUBSCRIBE_CHANNEL_STATUS; // 11021
	msg.m_ui4_recv_office = BROADCAST_MAIL_SLOT;
	msg.m_ui4_recv_slot = POST_OFFICE_MAIL_SLOT;
	msg.m_ui4_send_office = m_sp_mail_slot->cr_post_office_id;
	msg.m_ui4_send_slot = m_sp_mail_slot->cr_mail_slot_id;

    c_memory_iostream ios(true);
    ios.out((UI4) 60 * 1000);
	
	msg.m_data = ios.flash_to_memory();
	int ret = m_sp_mail_slot->send(&msg,1);
    if (ret != 1) {
        printf("subscribeFeChannelStatus send failed, ret:%d\n", ret);
        return -1;
    }
    return 0;
}

int subscribeFeChannelLogs(uint32_t fechannelID, uint32_t pluginID) {
    if (!m_sp_mail_slot.valid()) {
        printf("subscribeFeChannelLogs m_sp_mail_slot is invalid\n");
        return -2;
    }
    c_mail msg;
    msg.m_ui4_type = FES::MT_SUBSCRIBE_DISPLAY_CODE;
    msg.m_ui4_recv_office = BROADCAST_MAIL_SLOT;
    msg.m_ui4_recv_slot = (UI4)fechannelID;
    msg.m_ui4_send_office = m_sp_mail_slot->cr_post_office_id;
    msg.m_ui4_send_slot = m_sp_mail_slot->cr_mail_slot_id;
    c_key key((UI4)pluginID); // 插件id
    key.m_ui2_class_id = pluginID; // 插件id
    c_memory_iostream ios(true);
    ios.out(key);
    msg.m_data = ios.flash_to_memory();
    int ret = m_sp_mail_slot->send(&msg,1); // 默认1
    if (ret != 1) {
        printf("subscribeFeChannelLogs send failed, ret:%d\n", ret);
        return -1;
    }
    return 0;
}
//
// int notifyFeChannelReload() {
//     if (!m_sp_mail_slot.valid()) {
//         printf("notifyFeChannelReload m_sp_mail_slot is invalid\n");
//         return -2;
//     }
//     c_mail msg;
// 	msg.m_ui4_recv_office = m_sp_mail_slot->cr_post_office_id;
//     msg.m_ui4_recv_slot = POST_OFFICE_MAIL_SLOT;
//     msg.m_ui4_send_office = m_sp_mail_slot->cr_post_office_id;
//     msg.m_ui4_send_slot = m_sp_mail_slot->cr_mail_slot_id;
//     msg.m_ui4_type = FES::MT_RELOAD;
//     msg.m_ui4_qualify = 0 ;
//     c_memory_iostream ios ;
//     ios.out((UI4)-1) ;
//     msg.m_data = ios.flash_to_memory() ;
//
// 	int ret = m_sp_mail_slot->send(&msg,1);
//     if (ret != 1) {
//         printf("notifyFeChannelReload send failed, ret:%d\n", ret);
//         return -1;
//     }
//     return 0;
// }
int notifyFeServerReload(uint32_t channel_ids[],int size)
{
    if (!m_sp_mail_slot.valid())
        return 1 ;

    c_mail msg ;
    msg.m_ui4_recv_office = m_sp_mail_slot->cr_post_office_id ;
    msg.m_ui4_recv_slot   = POST_OFFICE_MAIL_SLOT ;
    msg.m_ui4_send_office = m_sp_mail_slot->cr_post_office_id ;
    msg.m_ui4_send_slot   = m_sp_mail_slot->cr_mail_slot_id ;
    msg.m_ui4_type        = FES::MT_RELOAD ;
    msg.m_ui4_qualify     = 0 ;

    c_memory_iostream ios ;
    if (size == 0)
    {
        ios.out((UI4)-1) ; // 0xffffffff 表示全部通道重载
    }
    else
    {
        for (size_t i = 0 ; i < size ; i ++)
            ios.out(channel_ids[i]) ;
    }
    msg.m_data = ios.flash_to_memory() ;

    if (m_sp_mail_slot->send(&msg,1) != 1)
        return 1 ;

    return 0 ;
}

namespace {
    Poco::JSON::Object::Ptr handleDisplayCode(const c_mail& mail) {
        c_memory_iostream ios(
            reinterpret_cast<UI1*>(mail.m_data->cr_point),
            mail.m_data->cr_size
        );

        c_key rkey;
        UI4 clr;
        std::string s;

        if (!ios.in(rkey)) {
            printf("recvFeChannelMsg: DISPLAY_CODE read rkey failed\n");
            return nullptr;
        }
        if (!ios.in(clr)) {
            printf("recvFeChannelMsg: DISPLAY_CODE read clr failed\n");
            return nullptr;
        }
        if (!ios.in(s)) {
            printf("recvFeChannelMsg: DISPLAY_CODE read string failed\n");
            return nullptr;
        }

        Poco::JSON::Object::Ptr root = new Poco::JSON::Object();
        root->set("channelid", static_cast<UI4>(mail.m_ui4_send_slot));
        root->set("pluginid", static_cast<UI4>(rkey.m_ui2_class_id));
        root->set("loginfo", s);

//         printf("recvFeChannelMsg DISPLAY_CODE classid:%ld，m_ui2_instance_id：%ld, channelid:%ld, msg:%s\n",
//                rkey.m_ui2_class_id, rkey.m_ui2_instance_id, mail.m_ui4_send_slot, s.c_str());

        return root;
    }

    Poco::JSON::Object::Ptr handleChannelStatus(const c_mail& mail) {
        c_memory_iostream ios(mail.m_data);  // 假设构造函数正确

        Poco::JSON::Array::Ptr statusArr = new Poco::JSON::Array();
        while (ios.size() >= 7) {
            UI4 channelId;
            UI1 duty, connected, used;

            if (!ios.in(channelId)) {
                printf("recvFeChannelMsg: CHANNEL_STATUS read channel_id failed\n");
                return nullptr;
            }
            if (!ios.in(duty)) {
                printf("recvFeChannelMsg: CHANNEL_STATUS read duty failed\n");
                return nullptr;
            }
            if (!ios.in(connected)) {
                printf("recvFeChannelMsg: CHANNEL_STATUS read connected failed\n");
                return nullptr;
            }
            if (!ios.in(used)) {
                printf("recvFeChannelMsg: CHANNEL_STATUS read used failed\n");
                return nullptr;
            }

            // printf("recvFeChannelMsg CHANNEL_STATUS channel_id:%llu, duty:%d, connected:%d, used:%d\n",
            //     static_cast<unsigned long long>(channelId), duty, connected, used);

            Poco::JSON::Object::Ptr item = new Poco::JSON::Object();
            item->set("channelid", channelId);
            item->set("duty", duty);
            item->set("connected", connected);
            item->set("used", used);

            statusArr->add(item);
        }

        Poco::JSON::Object::Ptr root = new Poco::JSON::Object();
        root->set("status_list", statusArr);

        return root;
    }

    Poco::JSON::Object::Ptr handleReloadResult(const c_mail& mail) {
        UI4 result = static_cast<UI4>(-1);
        c_memory_iostream ios(
            reinterpret_cast<UI1*>(mail.m_data->cr_point),
            mail.m_data->cr_size
        );

        if (!ios.in(result)) {
            printf("recvFeChannelMsg: RELOAD_RESULT read result failed\n");
            return nullptr;
        }

        // printf("recvFeChannelMsg reload result: %d\n", result);

        Poco::JSON::Object::Ptr root = new Poco::JSON::Object();
        root->set("result", result);

        return root;
    }
}

int recvFeChannelMsg(char** outputJson) {
    // 初始化输出指针
    if (outputJson) *outputJson = nullptr;

    // 1. 检查消息槽有效性
    if (!m_sp_mail_slot.valid()) {
        printf("recvFeChannelMsg: mail slot is invalid\n");
        return -2;
    }

    // 2. 接收消息
    c_mail mailBuffer[1];
    int n = m_sp_mail_slot->recv(mailBuffer, 1);
    if (n <= 0) {
        // 无邮件或接收失败
        return n;
    }

    // 3. 准备 JSON 数组
    Poco::JSON::Array::Ptr messageObj = nullptr;

    // 4. 遍历处理消息内容
    for (int i = 0; i < n; ++i) {
        const c_mail& mail = mailBuffer[i];
        // printf("recv mail: %d, type: %lld, idx: %d\n", n, mail.m_ui4_type, i);

        Poco::JSON::Object::Ptr messageObj = nullptr;

        switch (mail.m_ui4_type) {
            case FES::MT_DISPLAY_CODE:      // 11003
                messageObj = handleDisplayCode(mail);
                break;
            case FES::MT_SUBSCRIBE_CHANNEL_STATUS:  // 11021
                messageObj = handleChannelStatus(mail);
                break;
            case FES::MT_RELOAD_RESULT:         // 11011
                messageObj = handleReloadResult(mail);
                break;
            default:
                // 未知消息类型，忽略
                printf("recvFeChannelMsg: unknown message type %lld\n", mail.m_ui4_type);
                continue;
        }

        if (messageObj) {
            // 4. 将 JSON 对象序列化为字符串
            std::ostringstream oss;
            messageObj->stringify(oss);
            std::string jsonStr = oss.str();

            char* output = static_cast<char*>(malloc(jsonStr.length() + 1));
            if (!output) {
                printf("recvFeChannelMsg: malloc failed\n");
                return -1;
            }
            strcpy(output, jsonStr.c_str());
            *outputJson = output;

            // 5. 返回成功处理的消息类型
            return mail.m_ui4_type;
        } else {
            return -1;
        }
    }
}

