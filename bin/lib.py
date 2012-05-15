
import optparse
import os
import subprocess

def GetProjectRoot():
  return os.path.abspath(os.path.join(
    os.path.dirname(__file__),
    os.path.pardir))

def _UpdateGitDep(deps, *urls):
  for url in urls:
    name = url[url.rindex('/') + 1:]
    if name.endswith('.git'):
      name = name[:-4]
    if not _UpdateGitClient(os.path.join(GetProjectRoot(), deps, name), url):
      return False
  return True

def _UpdateGitClient(path, url):
  if not os.path.exists(path):
    return subprocess.call(['git', 'clone', url, path]) == 0
  else:
    return subprocess.call(['git', 'pull'], cwd = path) == 0

def _DownloadTar(url, dst):
  a = subprocess.Popen(['curl', '--silent', url], stdout = subprocess.PIPE)
  b = subprocess.Popen(['tar', 'zxvf', '-'], stdin = a.stdout, cwd = dst)
  a.stdout.close()
  return b.wait() == 0

def _Go(goroot, gopath, args):
  env = os.environ
  env['GOPATH'] = ':'.join(gopath)
  return subprocess.call([os.path.join(goroot, 'bin/go')] + args,
    env = env) == 0

def _OptionParser():
  parser = optparse.OptionParser()
  parser.add_option('--goroot',
    dest = 'goroot',
    default = '/usr/local/go',
    help = '')
  return parser

def Build():
  options, args = _OptionParser().parse_args()

  root = GetProjectRoot()
  deps = os.path.join(root, 'deps')
  outs = os.path.join(root, 'outs')
  if not os.path.exists(deps) or not os.path.exists(outs):
    lib.Setup()

  _Go(os.path.abspath(options.goroot),
    [outs, root],
    ['build', '-o', 'pork', 'src/pork-cli.go'])

def Setup():
  options, args = _OptionParser().parse_args()

  root = GetProjectRoot()
  deps = os.path.join(root, 'deps')
  if not os.path.exists(deps):
    os.makedirs(deps)
  
  jsc = os.path.join(deps, 'closure')
  if not os.path.exists(jsc):
    os.makedirs(jsc)

  outs = os.path.join(root, 'outs')
  if not os.path.exists(outs):
    os.makedirs(outs)

  # todo: remove this code, this is getting checked in directly.
  # if not _DownloadTar('http://closure-compiler.googlecode.com/files/compiler-latest.tar.gz', jsc):
  #   return False

  if not _UpdateGitDep(deps, 'git://github.com/nex3/sass.git'):
    return False

  if not _Go(os.path.abspath(options.goroot),
      [outs, root],
      ['get', 'github.com/hoisie/mustache']):
    return False
