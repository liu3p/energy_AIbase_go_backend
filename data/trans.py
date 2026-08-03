import csv
import json

# 读取 CSV 文件
csv_file_path = 'lansource.csv'  # 替换为你的 CSV 文件路径
md_csv_file_path = 'lansource_modbus.csv'  # 替换为你的 CSV 文件路径

# 生成国际化 JSON 文件
clientlan_json = {"version": "1", "content": {"zh": {}, "en": {}}}

# 生成单位与取值信息 JSON 文件
dataunit_json = {"version": "1", "content": {}}

# 生成modbus转发数据的国际化 JSON 文件
clientlan_mb_json = {}

# 生成modbus转发数据的生成单位与取值信息 JSON 文件
dataunit_mb_json = {}


# 定义函数来处理每行数据
def process_row(device_code, point):
    if not device_code and point:  # 如果设备编码为空且点位有值
        return point
    elif device_code and point:  # 如果两列都有值
        return f"{device_code}_{point}"
    else:  # 其他情况（比如两列都为空）
        return None  # 或者你可以返回空字符串或其他默认值

with open(csv_file_path, 'r', encoding='utf-8') as csv_file:
    csv_reader = csv.DictReader(csv_file)
    for row in csv_reader:
        #key = f"{row['设备编码']}_{row['点位']}"
        device_code = row.get('设备编码')
        point = row.get('点位')
        key = process_row(device_code, point)
        if key is None:
            print(f"empty line? row:{row}")
            continue
        
        # 生成国际化 JSON
        if row['点位中文'] != "" or row['点位英文'] != "":
            clientlan_json["content"]["zh"][key] = row['点位中文']
            clientlan_json["content"]["en"][key] = row['点位英文']

        # 生成单位与取值信息 JSON
        entry={}
        if row['单位']:
            entry["unit"] = row['单位']
        if row['取值']:
            values = {}
            value_pairs = row['取值'].split('；')
            for pair in value_pairs:
                value, description = pair.split('-')
                values[value] = f"{key}_val_{value}"
            entry["values"] = values
        if entry:
            dataunit_json["content"][key] = entry

with open(md_csv_file_path, 'r', encoding='utf-8') as csv_file:
    csv_reader = csv.DictReader(csv_file)
    for row in csv_reader:
        device_code = row.get('设备型号')
        point = row.get('点位')

        if device_code not in clientlan_mb_json:
            clientlan_mb_json[device_code] = {"zh":{}, "en":{}}

        if row['点位中文'] != "" or row['点位英文'] != "":
            clientlan_mb_json[device_code]["zh"][point] = row['点位中文']
            clientlan_mb_json[device_code]["en"][point] = row['点位英文']

        entry={}
        if row['单位']:
            entry["unit"] = row['单位']
        if entry:
            if device_code not in dataunit_mb_json:
                dataunit_mb_json[device_code] = {}
            dataunit_mb_json[device_code][point] = entry


with open('dataunit.txt', 'w') as dataunit_file:
    json.dump(dataunit_json, dataunit_file, indent=2, ensure_ascii=False)

with open('clientlan.txt', 'w') as clientlan_file:
    json.dump(clientlan_json, clientlan_file, indent=2, ensure_ascii=False)

with open('dataunit_mb.txt', 'w') as dataunit_file:
    json.dump(dataunit_mb_json, dataunit_file, indent=2, ensure_ascii=False)

with open('clientlan_mb.txt', 'w') as clientlan_file:
    json.dump(clientlan_mb_json, clientlan_file, indent=2, ensure_ascii=False)

