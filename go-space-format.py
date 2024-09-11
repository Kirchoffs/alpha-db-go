###################################
##### I don't like tabs in Go #####
###################################

import os

def replace_tabs_with_spaces(file_path):
    with open(file_path, 'r') as f:
        contents = f.read()
    contents = contents.replace('\t', '    ')
    with open(file_path, 'w') as f:
        f.write(contents)

def replace_tabs_in_folder(folder_path):
    for dirpath, dirnames, filenames in os.walk(folder_path):
        for filename in filenames:
            if filename.endswith('.go'):
                file_path = os.path.join(dirpath, filename)
                print(file_path)
                replace_tabs_with_spaces(file_path)

replace_tabs_in_folder(os.getcwdb().decode('utf-8'))
