s/\(^.*yyS := .*$\)/\1\
\
        startyyVAL := yyVAL\
start:\
        yyVAL = startyyVAL/g
s/\(^		yychar = yylex1(yylex, &yylval)$\)/\1\
		if yychar == yyTok2[ERROR-yyPrivate] {\
			goto start\
		}\
/g
s/\(^			yychar = yylex1(yylex, &yylval)$\)/\1\
			if yychar == yyTok2[ERROR-yyPrivate] {\
				goto start\
			}\
/g
