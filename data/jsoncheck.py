import json

def check_json_legality(file_path):
    try:
        # 打开文件并加载 JSON 数据
        with open(file_path, 'r', encoding='utf-8') as f:
            json.load(f)
        print(f"{file_path} 中的 JSON 格式合法")
    except json.JSONDecodeError as e:
        print(f"{file_path} 中的 JSON 格式不合法: {e}")
    except FileNotFoundError:
        print(f"{file_path} 文件未找到")

# 使用示例
file_path = 'dataunit.txt'  # 替换为你的文件路径
check_json_legality(file_path)

file_path = 'clientlan.txt'  # 替换为你的文件路径
check_json_legality(file_path)
