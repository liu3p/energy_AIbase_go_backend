#include "utility/product_line.h"
#include "utility/t_link.h"
#include <unistd.h> 
#include <signal.h>
#include "stdio.h"

#include "fe_model/fe_model_manage.h"

int main()
{
    FES::load_all_model_and_plugin();

    TFC_UTILITY::t_set<UI4> fac_ids;
    c_work_shop_factory::enum_factory_ids(fac_ids);

    for(int index=0;index<fac_ids.size();index++)
    {
        UI4 id = fac_ids.get_data(index);
        TFC_UTILITY::t_link<TFC_UTILITY::c_parameter_description *> all_param ;
        TFC_UTILITY::i_worker_factory * p_wf = c_work_shop_factory::find_factory(id) ;
        if(p_wf == NULL) 
        {
            printf("find factory id:%d NULL!\n",id);
            continue;
        }
        p_wf->all_parameter_description(all_param);

        printf("factory id:%ld,name:%s\n",id,p_wf->get_name().c_str());

        TFC_UTILITY::c_handle h = all_param.first_node() ;
        for(I4 i = 0 ; i < all_param.size() && HANDLE_VALID(h) ; i ++ , h = all_param.get_next_handle(h))
        {
            TFC_UTILITY::c_parameter_description * p_param_desp = NULL ;
            make_sure(all_param.get_data(h,p_param_desp)) ;

            if(p_param_desp == NULL ) return;

            printf("\t%s:%s\n", p_param_desp->m_str_name.c_str(),p_param_desp->m_str_default_value.c_str());
        }
    }

    FES::unload_all_model_and_plugin() ;

    return 0;
}