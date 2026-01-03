#!/usr/bin/env python3
import argparse
import os
import subprocess
import sys
from urllib.parse import urlparse


def create_parser(ops: dict[str, str]):
	parser = argparse.ArgumentParser(
		prog='mpc',
		description='Wrapper for running commands through mp connect proxy',
		formatter_class=argparse.RawDescriptionHelpFormatter,
		epilog=f"""
supported commands:
  {'\n  '.join(f'{protocol}\t\t{ops[protocol]}' for protocol in ops)}

examples:
  mpc ssh user@host
  mpc scp file.txt user@host:/path/
  mpc curl https://api.example.com/endpoint
  mpc git clone git@github.com:user/repo.git
  mpc mysql -h dbhost -u user -p
  mpc psql -h dbhost -U user dbname
  mpc redis-cli -h redishost
		"""
	)
	parser.add_argument(
		'command',
		choices=[protocol for protocol in ops],
		help='command to run through mp connect'
	)
	parser.add_argument(
		'args',
		nargs=argparse.REMAINDER,
		help='arguments to pass to the command'
	)
	return parser

def extract_host_from_args(args, flag='-h'):
	"""Extract host from command arguments like -h hostname or --host=hostname"""
	for i, arg in enumerate(args):
		if arg == flag and i + 1 < len(args):
			return args[i + 1]
		if arg.startswith(f'{flag}='):
			return arg.split('=', 1)[1]
		if flag == '-h' and arg.startswith('--host='):
			return arg.split('=', 1)[1]
	return None

def handle_http_proxy_command(cmd, args):
	"""Handle curl/wget/httpie which use HTTP(S) proxies"""
	# Find the URL from arguments
	url = None
	for arg in args:
		if arg.startswith('http://') or arg.startswith('https://'):
			url = arg
			break

	if not url:
		print(f"Error: No URL found in {cmd} command", file=sys.stderr)
		sys.exit(1)

	# Parse URL to get protocol and port
	parsed_url = urlparse(url)
	protocol = parsed_url.scheme
	port = parsed_url.port

	# Default ports if not specified
	if port is None:
		port = 443 if protocol == 'https' else 80

	# Start mp connect in background
	mp_process = subprocess.Popen(
		["mp", "connect", "-o", protocol, str(port)],
		stdout=subprocess.PIPE,
		stderr=subprocess.PIPE,
	)

	if not mp_process.stdout:
		raise Exception("could not open stdout of mp connect.")

	# Read the output port (assuming it outputs as 4 bytes)
	bytes_data = mp_process.stdout.read(4)
	mp_port = int.from_bytes(bytes_data, byteorder='big', signed=False)

	try:
		# Run command through the proxy
		proxy_url = f"{protocol}://localhost:{mp_port}"

		if cmd == "curl":
			result = subprocess.run(["curl", "--proxy", proxy_url] + args)
		elif cmd == "wget":
			# wget uses different proxy flags
			env = os.environ.copy()
			if protocol == "https":
				env["https_proxy"] = proxy_url
			else:
				env["http_proxy"] = proxy_url
			result = subprocess.run(["wget"] + args, env=env)
		elif cmd == "http" or cmd == "httpie":
			# httpie uses --proxy flag
			result = subprocess.run(["http", f"--proxy={protocol}:{proxy_url}"] + args)
		else:
			result = subprocess.run([cmd] + args)

		sys.exit(result.returncode)
	finally:
		# Kill the mp connect process
		mp_process.terminate()
		mp_process.wait()

def handle_tcp_command(cmd, args, default_port):
	"""Handle database/service commands that connect via TCP"""
	# Extract hostname from arguments
	host = extract_host_from_args(args, '-h')

	if not host:
		print(f"Error: No host found in {cmd} command (use -h hostname)", file=sys.stderr)
		sys.exit(1)

	# Extract port if specified
	port = None
	for flag in ['-p', '--port', '-P']:
		port_str = extract_host_from_args(args, flag)
		if port_str:
			try:
				port = int(port_str)
			except ValueError:
				pass
			break

	if port is None:
		port = default_port

	# Start mp connect in background
	mp_process = subprocess.Popen(
		["mp", "connect", "-o", "tcp", str(port)],
		stdout=subprocess.PIPE,
		stderr=subprocess.PIPE,
	)

	if not mp_process.stdout:
		raise Exception("could not open stdout of mp connect.")

	# Read the output port
	bytes_data = mp_process.stdout.read(4)
	mp_port = int.from_bytes(bytes_data, byteorder='big', signed=False)

	try:
		# Replace the host in arguments with localhost:mp_port
		new_args = []
		skip_next = False
		for i, arg in enumerate(args):
			if skip_next:
				skip_next = False
				new_args.append(str(mp_port))
				continue

			if arg == '-h':
				new_args.append(arg)
				new_args.append('localhost')
				skip_next = True
			elif arg.startswith('-h=') or arg.startswith('--host='):
				flag, _ = arg.split('=', 1)
				new_args.append(f"{flag}=localhost")
			elif arg == '-p' or arg == '-P' or arg == '--port':
				new_args.append(arg)
				skip_next = True
			else:
				new_args.append(arg)

		result = subprocess.run([cmd] + new_args)
		sys.exit(result.returncode)
	finally:
		mp_process.terminate()
		mp_process.wait()

def main():
	parser = create_parser({
		'ssh': 'SSH connection through mp connect',
		'scp': 'Secure copy through mp connect',
		'sftp': 'SFTP connection through mp connect',
		'rsync': 'Rsync through mp connect',
		'git': 'Git operations through mp connect',
		'curl': 'Curl requests through mp connect proxy',
		'wget': 'Wget requests through mp connect proxy',
		'httpie': 'HTTPie requests through mp connect proxy',
		'mysql': 'MySQL connection through mp connect',
		'psql': 'PostgreSQL connection through mp connect',
		'redis-cli': 'Redis CLI through mp connect',
		'mongosh': 'MongoDB shell through mp connect',
		'xfreerdp': 'RDP connection through mp connect',
		'vncviewer': 'VNC viewer through mp connect',
		'ldapsearch': 'LDAP search through mp connect',
		'ftp': 'FTP connection through mp connect',
	})

	parsed = parser.parse_args()
	cmd = parsed.command
	args = parsed.args

	# SSH-based commands
	if cmd == "ssh":
		result = subprocess.run(
			["ssh", "-o", "ProxyCommand=mp connect ssh %h %p"] + args
		)
		sys.exit(result.returncode)

	elif cmd == "scp":
		result = subprocess.run(
			["scp", "-o", "ProxyCommand=mp connect scp %h %p"] + args
		)
		sys.exit(result.returncode)

	elif cmd == "sftp":
		result = subprocess.run(
			["sftp", "-o", "ProxyCommand=mp connect ssh %h %p"] + args
		)
		sys.exit(result.returncode)

	elif cmd == "rsync":
		result = subprocess.run(
			["rsync", "-e", "ssh -o ProxyCommand='mp connect rsync %h %p'"] + args
		)
		sys.exit(result.returncode)

	elif cmd == "git":
		env = os.environ.copy()
		env["GIT_SSH_COMMAND"] = "ssh -o ProxyCommand='mp connect git %h %p'"
		result = subprocess.run(["git"] + args, env=env)
		sys.exit(result.returncode)

	# HTTP(S) proxy commands
	elif cmd in ["curl", "wget", "httpie"]:
		handle_http_proxy_command(cmd, args)

	# Database commands
	elif cmd == "mysql":
		handle_tcp_command(cmd, args, 3306)

	elif cmd == "psql":
		handle_tcp_command(cmd, args, 5432)

	elif cmd == "redis-cli":
		handle_tcp_command(cmd, args, 6379)

	elif cmd == "mongosh":
		handle_tcp_command(cmd, args, 27017)

	elif cmd == "ldapsearch":
		handle_tcp_command(cmd, args, 389)

	elif cmd == "ftp":
		handle_tcp_command(cmd, args, 21)

	# RDP/VNC commands
	elif cmd == "xfreerdp":
		handle_tcp_command(cmd, args, 3389)

	elif cmd == "vncviewer":
		handle_tcp_command(cmd, args, 5900)

if __name__ == "__main__":
	main()
