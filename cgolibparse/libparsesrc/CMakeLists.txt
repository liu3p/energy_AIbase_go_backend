cmake_minimum_required(VERSION 3.16)
project(parse VERSION 1.0.0 LANGUAGES CXX)

# 设置C++标准
#set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# 检查平台 - 仅支持 Linux
if(NOT UNIX)
    message(FATAL_ERROR "This project only supports Linux platform")
endif()

add_compile_options(-Wall -Wextra -Wno-unused-parameter -g)
add_compile_options(-fpermissive)
add_compile_options(-w)
add_compile_options(-fPIC)

include_directories(
    $ENV{ARM_SF}/src/util/include
    $ENV{ARM_SF}/src/fe/fe_ems_extend/include
    $ENV{ARM_SF}/src/fe/fe_kernel/include
    $ENV{ARM_SF}/src/translation/include
    $ENV{POCO_ROOT}/include
    $ENV{ACE_ROOT}
)


# 源文件列表
set(SOURCES
  cgo_pluginparse.cpp
)

# 
#add_executable(parameter ${SOURCES} ${HEADERS})
add_library(parse SHARED ${SOURCES}) 

target_include_directories(parse
PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}/.. 
PUBLIC $ENV{ARM_SF}/src/agc/include/
PUBLIC $ENV{ARM_SF}/src/fe/fe_kernel/include/ # 公共头文件路径
PUBLIC $ENV{ARM_SF}/src/util/include/
$ENV{ARM_SF}/src/fe/fe_ems_extend/include
$ENV{POCO_ROOT}/include
$ENV{ACE_ROOT}
$ENV{ARM_SF}/src/fe/fe_kernel/include
$ENV{ARM_SF}/src/fe/fe_kernel/fe_server/src/
)

# 设置输出目录
set_target_properties(parse PROPERTIES
    ARCHIVE_OUTPUT_DIRECTORY $ENV{ARM_SF}/lib
    LIBRARY_OUTPUT_DIRECTORY $ENV{ARM_SF}/lib
    RUNTIME_OUTPUT_DIRECTORY $ENV{ARM_SF}/bin
)

install(TARGETS parse
    LIBRARY DESTINATION lib          # 安装到 ${CMAKE_INSTALL_PREFIX}/lib
)

# 链接目录
target_link_directories(parse PRIVATE
    $ENV{ARM_SF}/lib
)


# Linux 特定设置
target_link_libraries(parse agc_base pthread rt dl utility fe_model_manage PocoJSON PocoNet PocoFoundation PocoXML fe_api sflog)

# # 设置中间文件目录
# set_target_properties(parse PROPERTIES
#     ARCHIVE_OUTPUT_DIRECTORY ./lib
#     LIBRARY_OUTPUT_DIRECTORY ./lib
#     RUNTIME_OUTPUT_DIRECTORY ./bin
# )

set(CMAKE_LIBRARY_OUTPUT_DIRECTORY $ENV{ARM_SF}/lib)

# 环境变量检查
if(NOT DEFINED ENV{POCO_ROOT})
    message(WARNING "POCO_ROOT environment variable is not set")
endif()

