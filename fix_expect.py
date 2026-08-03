import os
import re

directory = 'internal/application/'
for filename in os.listdir(directory):
    if filename.endswith('_test.go'):
        filepath = os.path.join(directory, filename)
        with open(filepath, 'r') as f:
            content = f.read()

        # Replace .EXPECT()."MethodName", with .EXPECT().MethodName(
        content = re.sub(r'\.EXPECT\(\)\."([^"]+)",\s*', r'.EXPECT().\1(', content)
        # Handle cases where there are no arguments: .EXPECT()."MethodName")
        content = re.sub(r'\.EXPECT\(\)\."([^"]+)"\)', r'.EXPECT().\1()', content)

        with open(filepath, 'w') as f:
            f.write(content)
