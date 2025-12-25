#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

mkdir -p "$SCRIPT_DIR/bin"
cd $SCRIPT_DIR
go build -o "$SCRIPT_DIR/bin/mp"

cat <<EOF > "$SCRIPT_DIR/bin/mpc"
#!/bin/bash

# add extra protocols here
case \$2 in
	"ssh")
		ssh -o ProxyCommand="mp --protocol=ssh --host=%h --port=\$1 connect" \${@:3}
		;;
	*)
		echo "invalid protocol '\$2'"
		exit 1
		;;
esac
EOF

chmod +x "$SCRIPT_DIR/bin/mpc"
