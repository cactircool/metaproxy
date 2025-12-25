import os
import stat
import subprocess
from pathlib import Path


def main(proxy_file: str, wrapper_file: str):
	script_dir: Path = Path(__file__).resolve().parent

	subprocess.run(['go', 'build', '-o', f'/usr/local/bin/{proxy_file}', script_dir.as_posix()], check=True)
	wrapper_file = f'/usr/local/bin/${wrapper_file}'
	with open(wrapper_file, 'w') as file:
		file.write(
			'''
			#!/bin/bash

			# add extra protocols here
			case $1 in
				"ssh")
					ssh -o ProxyCommand="mp --protocol=ssh --host=%h --port=%p connect" ${@:2}
					;;
				*)
					echo "invalid protocol '$1'"
					exit 1
					;;
			esac
			'''
		)

	current_perms = os.stat(wrapper_file).st_mode
	new_perms = current_perms | stat.S_IEXEC
	os.chmod(wrapper_file, new_perms)

	print("run 'mp' for the server and proxy engine.\nrun 'mpc' for the client connection engine.")

if __name__ == '__main__':
	proxy_file = input('what do you what to call the proxy and server engine? (default=mp) ').strip() or 'mp'
	wrapper_file = input('what do you want to call the client connection engine? (default=mpc) ').strip() or 'mpc'
	main(proxy_file, wrapper_file)
