s/\(^.*yyS := .*$\)/\1\
\
	startyyVAL := yyVAL\
start:\
	yyVAL = startyyVAL/g
s/\(^		yyrcvr\.char, yytoken = yylex1(yylex, &yyrcvr\.lval)$\)/\1\
		if yyrcvr.char == CTRLC {\
			goto start\
		}\
/g
s/\(^			yyrcvr\.char, yytoken = yylex1(yylex, &yyrcvr\.lval)$\)/\1\
			if yyrcvr.char == CTRLC {\
				goto start\
			}\
/g
s/""\(.\{1,2\}\)""/"\1"/g
