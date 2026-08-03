cmake_minimum_required(VERSION 3.16)
project(parameter VERSION 1.0.0 LANGUAGES CXX)

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

# 源文件列表
set(SOURCES
  src/main.cpp
)

# 
add_executable(parameter ${SOURCES} ${HEADERS})

target_include_directories(parameter
PUBLIC $ENV{ARM_SF}/src/fe/fe_kernel/include/ # 公共头文件路径
PUBLIC $ENV{ARM_SF}/src/util/include/
$ENV{ARM_SF}/src/fe/fe_ems_extend/include
$ENV{POCO_ROOT}/include
$ENV{ACE_ROOT}
$ENV{ARM_SF}/src/fe/fe_kernel/include
$ENV{ARM_SF}/src/fe/fe_kernel/fe_server/src/
)

# 设置输出目录
set_target_properties(parameter PROPERTIES
    LIBRARY_OUTPUT_DIRECTORY $ENV{ARM_SF}/bin
    RUNTIME_OUTPUT_DIRECTORY $ENV{ARM_SF}/lib
)


# 链接目录
target_link_directories(parameter PRIVATE
    $ENV{ARM_SF}/lib
    /root/depend/POCO/lib
)



# Linux 特定设置
target_link_libraries(parameter  pthread rt dl utility fe_model_manage PocoNet PocoFoundation PocoXML fe_api sflog)

# 设置中间文件目录
set_target_properties(parameter PROPERTIES
    ARCHIVE_OUTPUT_DIRECTORY ./lib
    LIBRARY_OUTPUT_DIRECTORY ./lib
    RUNTIME_OUTPUT_DIRECTORY ./bin
)

# 安装目标
install(TARGETS parameter
    LIBRARY DESTINATION ./lib
    RUNTIME DESTINATION ./bin
)

# 环境变量检查
if(NOT DEFINED ENV{POCO_ROOT})
    message(WARNING "POCO_ROOT environment variable is not set")
endif()

