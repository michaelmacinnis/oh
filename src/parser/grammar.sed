s/\(^.*yyS := .*$\)/\1\
\
	startyyVAL := yyVAL\
start:\
	yyVAL = startyyVAL/g
s/\(^		yychar, yytoken = yylex1(yylex, &yylval)$\)/\1\
		if yychar == ERROR {\
			goto start\
		}\
/g
s/\(^			yychar, yytoken = yylex1(yylex, &yylval)$\)/\1\
			if yychar == ERROR {\
				goto start\
			}\
/g
s/""\(.\{1,2\}\)""/"\1"/g
