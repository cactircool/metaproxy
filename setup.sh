#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

mkdir -p "$SCRIPT_DIR/bin"
cd $SCRIPT_DIR
go build -o "$SCRIPT_DIR/bin/mp"

cat <<EOF > "$SCRIPT_DIR/bin/mpc"
#!/bin/bash

# add extra protocols here
case \$1 in
	"ssh")
		ssh -o ProxyCommand="mp --protocol=ssh --host=%h --port=%p" \${@:2}
		;;
	*)
		echo "invalid protocol '\$1'"
		exit 1
		;;
esac
EOF

chmod +x "$SCRIPT_DIR/bin/mpc"
