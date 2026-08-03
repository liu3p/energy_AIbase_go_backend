import json

def check_duplicate_keys(data, path=""):
    """
    递归检查 JSON 数据中是否有重复的键
    :param data: 要检查的 JSON 数据（可以是字典或列表）
    :param path: 当前 JSON 数据路径，用于定位重复键
    :return: 返回一个包含重复键路径的列表
    """
    if isinstance(data, dict):
        seen_keys = set()
        duplicate_keys = []
        for key, value in data.items():
            # 如果遇到重复的键，记录路径
            if key == "ess.pcs_IgbtT":
                print("get pcs_IgbtT:", value)
            if key in seen_keys:
                duplicate_keys.append(f"{path}/{key}")
                print("get pcs_IgbtT duplicate_keys")
            else:
                if key == "ess.pcs_IgbtT":
                    print("set pcs_IgbtT:", seen_keys, value)
                seen_keys.add(key)
            
            # 递归检查嵌套的字典或列表
            duplicate_keys.extend(check_duplicate_keys(value, path=f"{path}/{key}"))
            #print("duplicate_keys:",duplicate_keys)
        
        return duplicate_keys
    elif isinstance(data, list):
        duplicate_keys = []
        for idx, item in enumerate(data):
            duplicate_keys.extend(check_duplicate_keys(item, path=f"{path}/{idx}"))
        return duplicate_keys
    else:
        return []

def find_duplicate_keys_in_json(file_path):
    """
    检查 JSON 文件中是否有重复的键
    :param file_path: JSON 文件路径
    :return: 打印出重复键的路径（如果有的话）
    """
    with open(file_path, "r", encoding="utf-8") as f:
        try:
            data = json.load(f)  # 解析 JSON 文件
            duplicates = check_duplicate_keys(data)
            if duplicates:
                print("Found duplicate keys at the following paths:")
                for duplicate in duplicates:
                    print(duplicate)
            else:
                print("No duplicate keys found.")
        except json.JSONDecodeError as e:
            print(f"Error decoding JSON: {e}")

# 使用示例
file_path = "dataunit.txt"  # 将此路径替换为你的 JSON 文件路径
find_duplicate_keys_in_json(file_path)

