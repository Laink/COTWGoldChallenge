#!/bin/sh
# Compiles as3/ into internal/modclass/mod.swc, embedded in the program. Needs Apache Royale (ROYALE) and playerglobal.swc 10.2 (PLAYERGLOBAL).
set -e
PREVIEW=${PREVIEW:-false}
echo '<royale-config></royale-config>' > /tmp/empty-config.xml
"$ROYALE/bin/compc" -debug=false -load-config=/tmp/empty-config.xml -external-library-path="$PLAYERGLOBAL" \
	-define=CONFIG::preview,$PREVIEW -source-path=as3 -include-classes=COTWGoldChallengeMod,COTWGoldChallengeIcons -output="${OUT:-internal/modclass/mod.swc}"
