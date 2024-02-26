" Scan the following dirs recursively for tags
let g:project_tags_dirs = ['alatty', 'kittens', 'tools']
if exists('g:ale_linters')
    let g:ale_linters['python'] = ['mypy', 'ruff']
else
    let g:ale_linters = {'python': ['mypy', 'ruff']}
endif
let g:ale_python_mypy_executable = './mypy-editor-integration'
let g:ale_fixers = {
\       "python": ["ruff"],
\}
autocmd FileType python noremap \i :ALEFix<CR>
set wildignore+==template.py
set wildignore+=tags
set expandtab
set tabstop=4
set shiftwidth=4
set softtabstop=0
set smarttab
python3 <<endpython
import sys
sys.path.insert(0, os.path.abspath('.'))
import alatty
endpython
