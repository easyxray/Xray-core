#!/bin/bash
# 这个脚本用于在使用 go generate 生成 pb 文件后，自动丢弃那些只有版本号变化的 pb 文件的更改。

# 获取所有被修改的 pb 文件
modified_pb_files=$(git status --porcelain | awk '{if ($1 == "M" && $2 ~ /\.pb\.go$/) print $2}')

for file in $modified_pb_files; do
    # 获取文件的 diff 内容，忽略上下文行
    diff_output=$(git diff -U0 "$file" | grep -E "^[+-]" | grep -v -E "^(---|\+\+\+)")
    
    # 检查是否所有改动都只包含 protoc 版本相关信息
    has_real_changes=0
    
    while IFS= read -r line; do
        if [[ ! "$line" =~ "protoc" ]]; then
            has_real_changes=1
            break
        fi
    done <<< "$diff_output"
    
    if [ "$has_real_changes" -eq 0 ]; then
        echo "Reverting version-only changes in: $file"
        git checkout -- "$file"
    else
        echo "Keeping actual changes in: $file"
    fi
done

echo "Done."
