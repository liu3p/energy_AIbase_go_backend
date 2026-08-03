import re
from collections import defaultdict

def check_duplicate_keys_in_json(file_path):
    """
    检查 JSON 文件中的重复键
    :param file_path: JSON 文件路径
    :return: 打印重复键的路径（如果有的话）
    """
    key_pattern = re.compile(r'"([^"]+)":')  # 匹配 JSON 中的键 (key)
    duplicate_keys = defaultdict(list)  # 用于存储每个键出现的位置
    
    with open(file_path, "r", encoding="utf-8") as f:
        line_number = 1  # 当前行号
        for line in f:
            # 匹配每行中的键
            keys_in_line = key_pattern.findall(line)
            
            for key in keys_in_line:
                duplicate_keys[key].append(line_number)
            
            line_number += 1
    
    # 输出重复的键和它们出现的行号
    found_duplicates = False
    for key, lines in duplicate_keys.items():
        if len(lines) > 2:  # 如果某个键出现超过1次，说明它是重复的
            found_duplicates = True
            print(f"Duplicate key: '{key}' found at lines {', '.join(map(str, lines))}")

    if not found_duplicates:
        print("No duplicate keys found.")

# 使用示例
file_path = "clientlan.txt"  # 将此路径替换为你的 JSON 文件路径
check_duplicate_keys_in_json(file_path)

