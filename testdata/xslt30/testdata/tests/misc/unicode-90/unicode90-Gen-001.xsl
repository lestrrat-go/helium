<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="2.0">
    
    <xsl:output indent="yes" encoding="UTF-8" />
    
    <xsl:param name="charclass" select="'Ll'" />
    <xsl:variable name="negclass" select="concat('\P{', $charclass, '}')" />
    <xsl:variable name="posclass" select="concat('\p{', $charclass, '}')" />
    
    <xsl:param name="lowupperbound" select="55295 (: for proper tests, should be 55295 :)" />
    <xsl:param name="highupperbound" select="1114111 (: for proper tests, should be 1114111 :)" />
    
    <xsl:variable name="validrangeX" select="(9, 10, 13, 32 to $lowupperbound)" />  <!-- (9, 10, 13, 32 to 55295, 57344 to 65533, 65537 to 1114111) -->
    <xsl:variable name="validrange" select="(9, 10, 13, 32 to $lowupperbound, 57344 to 65533, 65537 to $highupperbound)" />  <!-- (9, 10, 13, 32 to 55295, 57344 to 65533, 65537 to 1114111) -->
    
    <xsl:param name="source" select="doc(concat('docs/unicode-', $charclass, '.xml'))" />
    
    <xsl:template name="start" match="/">
        <xsl:apply-templates select="$source" mode="fn-replace8" />
    </xsl:template>
    
    <xsl:template match="/" mode="fn-matches1">
        <!-- simple test, tests character itself with \p{X}-->
        <xsl:variable name="regex" select="$posclass" />
        <res charclass="{$charclass}" regex="{$regex}">
            <xsl:value-of select="distinct-values(*/c/matches(@c, $regex))" />
        </res>
    </xsl:template>
    
    <xsl:template match="/" mode="fn-matches2">
        <!-- simple test, tests character itself with [\p{X}]+ -->
        <xsl:variable name="regex" select="concat('[', $posclass, ']+')" />
        <res charclass="{$charclass}" regex="{$regex}">
            <xsl:value-of select="distinct-values(*/c/matches(@c, $regex))" />
        </res>
    </xsl:template>
        
    <xsl:template match="/" mode="fn-matches3">
        <!-- simple test, tests character itself with 'not in' expr [^\P{X}]-->
        <xsl:variable name="regex" select="concat('[^', $negclass, ']+')" />
        <res charclass="{$charclass}" regex="{$regex}">
            <xsl:value-of select="distinct-values(*/c/matches(@c, $regex))" />
        </res>
    </xsl:template>
    
    <xsl:template match="/" mode="fn-matches4">
        <!-- simple test, tests character itself with inverse test [&#x9;-&#x10FFFF;-[\P{X}]] -->
        <xsl:variable name="regex" select="concat('[&#x9;-&#x10FFFF;-[', $negclass, ']]')" />
        <res charclass="{$charclass}" regex="{$regex}">
            <xsl:value-of select="distinct-values(*/c/matches(@c, $regex))" />
        </res>
    </xsl:template>
    
    <xsl:template match="/" mode="fn-matches5">
        <!-- simple test, tests all characters not in this range with \P{X}, regex ^\P{X}+$ -->
        <xsl:variable name="regex" select="concat('^', $negclass, '+$')" />
        <res charclass="{$charclass}" regex="{$regex}">
            <xsl:variable name="c" select="*/c/@d/xs:integer(.)" />
            <xsl:variable name="allcharsnot" as="xs:integer*">
                <xsl:sequence select="($validrange[not(. = $c)])"/>                
            </xsl:variable>
            
            <xsl:value-of select="distinct-values(for $i in $allcharsnot return matches(codepoints-to-string($i), $regex))" />
        </res>
    </xsl:template>
        
    <xsl:template match="/" mode="fn-matches6">
        <!-- simple test, tests all characters not in this range with [\P{X}]+$ -->
        <xsl:variable name="regex" select="concat('[', $negclass, ']+$')" />
        <res charclass="{$charclass}" regex="{$regex}">
            <xsl:variable name="c" select="*/c/@d/xs:integer(.)" />
            <xsl:variable name="allcharsnot" as="xs:integer*">
                <xsl:sequence select="($validrange[not(. = $c)])"/>                
            </xsl:variable>
            
            <xsl:value-of select="distinct-values(for $i in $allcharsnot return matches(codepoints-to-string($i), $regex))" />
        </res>
    </xsl:template>
    
    <xsl:template match="/" mode="fn-matches7">
        <!-- simple test, tests all characters not in this range with inverse test like ^[^\p{X}]$ -->
        <xsl:variable name="regex" select="concat('^[^', $posclass, ']+$')" />
        <res charclass="{$charclass}" regex="{$regex}">
            <xsl:variable name="c" select="*/c/@d/xs:integer(.)" />
            <xsl:variable name="allcharsnot" as="xs:integer*">
                <xsl:sequence select="($validrange[not(. = $c)])"/>                
            </xsl:variable>
            
            <xsl:value-of select="distinct-values(for $i in $allcharsnot return matches(codepoints-to-string($i), $regex))" />
        </res>
    </xsl:template>
        
    <xsl:template match="/" mode="fn-matches8">
        <!-- simple test, tests all characters not in this range with inverse test like [&#x9;-&#x10FFFF;-[\p{X}]] -->
        <xsl:variable name="regex" select="concat('[&#x9;-&#x10FFFF;-[', $posclass, ']]')" />
        <res charclass="{$charclass}" regex="{$regex}">
            <xsl:variable name="c" select="*/c/@d/xs:integer(.)" />
            <xsl:variable name="allcharsnot" as="xs:integer*">
                <xsl:sequence select="($validrange[not(. = $c)])"/>                
            </xsl:variable>
            
            <xsl:value-of select="distinct-values(for $i in $allcharsnot return matches(codepoints-to-string($i), $regex))" />
        </res>
    </xsl:template>    
        
    <xsl:template match="/" mode="fn-matches9">
        <!-- Complementary test, tests that NO characters in full unicode range match with [\p{X}-[\p{X}]] -->
        <xsl:variable name="regex" select="concat('[', $posclass, '-[', $posclass, ']]')" />
        <res charclass="{$charclass}" regex="{$regex}">
            <xsl:value-of select="distinct-values(for $i in $validrange return not(matches(codepoints-to-string($i), $regex)))" />
        </res>
    </xsl:template>
    
    <xsl:template match="/" mode="fn-matches10">
        <!-- Complementary test, tests that ALL characters in full unicode range match with [\p{X}|[\P{X}] -->
        <xsl:variable name="regex" select="concat('[', $posclass, ']|[', $negclass, ']]')" />
        <res charclass="{$charclass}" regex="{$regex}">
            <xsl:value-of select="distinct-values(for $i in $validrange return matches(codepoints-to-string($i), '[\p{Ll}]|[\P{Ll}]'))" />
        </res>
    </xsl:template>
    
    <xsl:template match="/" mode="fn-matches11">
        <!-- whole string test: all \p{X} chars together must match with '^\p{X}*$' -->
        <xsl:variable name="regex" select="concat('^', $posclass, '*$')" />
        <res charclass="{$charclass}" regex="{$regex}">
            <xsl:value-of select="matches(string-join(*/c/@c, ''), $regex)" />
        </res>
    </xsl:template>  
    
    <xsl:template match="/" mode="fn-matches12">
        <!-- whole string test: all \P{X} chars together must match -->
        <xsl:variable name="regex" select="concat('^', $negclass, '*$')" />
        <res charclass="{$charclass}" regex="{$regex}">
            <xsl:variable name="c" select="*/c/@d/xs:integer(.)" />
            <xsl:variable name="allcharsnot" as="xs:string">
                <xsl:value-of select="codepoints-to-string($validrange[not(. = $c)])"/>                
            </xsl:variable>
            <xsl:value-of select="matches($allcharsnot, $negclass)" />
        </res>
    </xsl:template>
    
    <xsl:template match="/" mode="fn-matches13">
        <!-- character-by-character test for \p{X} use it to find invalid negative matches WITHIN the proper \p{X} range -->
        <xsl:variable name="regex" select="$posclass" />
        <res charclass="{$charclass}" regex="{$regex}">
            <xsl:variable name="result" as="element()*">
                <xsl:for-each select="*/c">
                    <xsl:if test="not(matches(@c, $regex))" >
                        <error hex="{@cp}" dec="{@d}" match-failed-for="{@c}" />
                    </xsl:if>
                </xsl:for-each>
            </xsl:variable>
            <xsl:value-of select="empty($result)[.]"/>
            <xsl:copy-of select="$result"/>
        </res>
    </xsl:template>
    
    <xsl:template match="/" mode="fn-matches14">
        <!-- character-by-character test for \p{X} use it to find invalid positive matches OUTSIDE the proper \p{X} range -->
        <xsl:variable name="regex" select="$posclass" />
        <res charclass="{$charclass}" regex="{$regex}">
            <xsl:variable name="c" select="*/c/@d/xs:integer(.)" />
            <xsl:variable name="allcharsnot" as="xs:integer*">
                <xsl:sequence select="$validrange[not(. = $c)]"/>                
            </xsl:variable>
            <xsl:variable name="result" as="element()*">
                <xsl:for-each select="$allcharsnot">
                    <xsl:if test="matches(codepoints-to-string(.), $regex) (: we expect no match :)" >
                        <error dec="{.}" match-failed-for="{.}" />
                    </xsl:if>
                </xsl:for-each>
            </xsl:variable>
            <xsl:value-of select="empty($result)[.]"/>
            <xsl:copy-of select="$result"/>
        </res>
    </xsl:template>
    
    <xsl:template match="/" mode="fn-matches15">
        <!-- character-by-character test for \P{X} (neg X) use it to find invalid negative matches WITHIN the proper \P{X} range -->
        <xsl:variable name="regex" select="$negclass" />
        <res charclass="{$charclass}" regex="{$regex}">
            <xsl:variable name="c" select="*/c/@d/xs:integer(.)" />
            <xsl:variable name="allcharsnot" as="xs:integer*">
                <xsl:sequence select="$validrange[not(. = $c)]"/>                
            </xsl:variable>
            <xsl:variable name="result" as="element()*">
                <xsl:for-each select="$allcharsnot">
                    <xsl:if test="not(matches(codepoints-to-string(.), $regex))">
                        <error dec="{.}" match-failed-for="{codepoints-to-string(.)}" />
                    </xsl:if>
                </xsl:for-each>
            </xsl:variable>
            <xsl:value-of select="empty($result)[.]"/>
            <xsl:copy-of select="$result"/>
        </res>
    </xsl:template>
    
    <xsl:template match="/" mode="fn-matches16">
        <!-- character-by-character test for \P{X} (neg X) use it to find invalid positive matches OUTSIDE the proper \P{X} range -->
        <xsl:variable name="regex" select="$negclass" />
        <res charclass="{$charclass}" regex="{$regex}">
            <xsl:variable name="result" as="element()*">
                <xsl:for-each select="*/c">
                    <xsl:if test="matches(@c, $regex) (:we expect no match :)">
                        <error hex="{@cp}" dec="{@d}" match-failed-for="{@c}"/>
                    </xsl:if>
                </xsl:for-each>
            </xsl:variable>
            <xsl:value-of select="empty($result)[.]"/>
            <xsl:copy-of select="$result"/>
        </res>
    </xsl:template>
    
    <xsl:template match="/" mode="fn-matches17">
        <!-- counting comparison for all matches \p{X}, using regex ^\p{X}+$  -->
        <xsl:variable name="regex" select="concat('^', $posclass, '+$')" />
        <res charclass="{$charclass}" regex="{$regex}">
            <xsl:variable name="actual-count" select="count(for $i in $validrange return matches(codepoints-to-string($i), $regex)[.])" />
            <xsl:variable name="expected-count" select="count(*/c/@c)" />
            <xsl:if test="$actual-count != $expected-count">
                <error actual-count="{$actual-count}" expected-count="{$expected-count}">Invalid count in \p{X} range</error>
            </xsl:if>
            <xsl:value-of select="($actual-count = $expected-count)[.]"/>
        </res>
    </xsl:template>
        
    <xsl:template match="/" mode="fn-matches18">
        <!-- counting comparison for all matches \P{X}, using regex ^\P{X}+$ -->
        <xsl:variable name="regex" select="concat('^', $negclass, '+$')" />
        <res charclass="{$charclass}" regex="{$regex}">
            <xsl:variable name="c" select="*/c/@d/xs:integer(.)" />
            <xsl:variable name="allcharsnot" as="xs:string*">
                <xsl:sequence select="for $i in $validrange[not(. = $c)] return codepoints-to-string($i)"/>                
            </xsl:variable>
            <xsl:variable name="actual-count" select="count(for $i in $validrange return matches(codepoints-to-string($i), $regex)[.])" />
            <xsl:variable name="expected-count" select="count($allcharsnot)" />
            <xsl:if test="$actual-count != $expected-count">
                <error actual-count="{$actual-count}" expected-count="{$expected-count}">Invalid count in \P{X} range</error>
            </xsl:if>
            <xsl:value-of select="($actual-count = $expected-count)[.]"/>
        </res>
    </xsl:template>
    
    <xsl:template match="/" mode="fn-matches19">
        <!-- more elaborate regex with \p{X} with some other characters, with capturing parens and backreference, reports failing strings -->
        <xsl:variable name="regex" select="replace('(%%%{1}X%%%{2,3})..([%%%][%%%]){2}Y\nZ\1', '%%%', concat('\\p{', $charclass, '}'))" />
        <res charclass="{$charclass}" regex="{$regex}">
            <xsl:variable name="result" as="element()*">
                <xsl:for-each select="*/c">
                    <xsl:variable name="chars" select="concat(@c, 'X', @c, @c, @c, '&#x10FFEE;&#x10FFEE;', @c, @c, @c, @c, 'Y&#xA;Z', @c, 'X', @c, @c, @c)" />
                    <xsl:if test="not(matches($chars, $regex))" >
                        <error hex="{@cp}" dec="{@d}" char="{@c}" match-failed-for="{$chars}" />
                    </xsl:if>
                </xsl:for-each>
            </xsl:variable>
            <xsl:value-of select="empty($result)[.]"/>
            <xsl:copy-of select="$result"/>
        </res>
    </xsl:template>
        
    <xsl:template match="/" mode="fn-matches20">
        <!-- more elaborate regex with \P{X} with some other characters, with capturing parens and backreference, reports failing strings -->
        <xsl:variable name="regex" select="replace('(%%%{1}X%%%{2,3})..([%%%][%%%]){2}Y\nZ\1', '%%%', concat('\\P{', $charclass, '}'))" />
        <res charclass="{$charclass}" regex="{$regex}">
            <xsl:variable name="result" as="element()*">
                <xsl:variable name="c" select="*/c/@d/xs:integer(.)" />
                <xsl:variable name="allcharsnot" as="xs:integer*">
                    <xsl:sequence select="$validrange[not(. = $c)]"/>                
                </xsl:variable>
                <xsl:for-each select="$allcharsnot">
                    <xsl:variable name="c" select="codepoints-to-string(.)"/>
                    <xsl:variable name="chars" select="concat($c, 'X', $c, $c, $c, '&#x10FFEE;&#x10FFEE;', $c, $c,  $c, $c, 'Y&#xA;Z', $c, 'X', $c, $c, $c)" />
                    <xsl:if test="not(matches($chars, $regex))" >
                        <error dec="{.}" char="{$c}" match-failed-for="{$chars}"/>
                    </xsl:if>
                </xsl:for-each>
            </xsl:variable>
            <xsl:value-of select="empty($result)[.]"/>
            <xsl:copy-of select="$result"/>
        </res>
    </xsl:template>
    
    <xsl:template match="/" mode="fn-matches21">
        <!-- try to match with ^\p{X}{Y}$, where Y is the total number of characters in this range -->
        <xsl:variable name="regex" select="concat('^', $posclass, '{', count(*/c), '}$')" />
        <res charclass="{$charclass}" regex="{$regex}">
            <xsl:value-of select="matches(string-join(*/c/@c, ''), $regex)" />
        </res>
    </xsl:template>
    
    <xsl:template match="/" mode="fn-matches22">
        <!-- try to match with \P{X}{Y}, where Y is the total number of characters outside this range -->
        <xsl:variable name="c" select="*/c/@d/xs:integer(.)" />
        <xsl:variable name="allcharsnot" as="xs:string*">
            <xsl:sequence select="for $i in $validrange[not(. = $c)] return codepoints-to-string($i)"/>                
        </xsl:variable>
        <xsl:variable name="regex" select="concat('^', $negclass, '{', count($allcharsnot), '}$')" />
        <res charclass="{$charclass}" regex="{$regex}">            
            <xsl:value-of select="matches(string-join($allcharsnot, ''), $regex)" />
        </res>
    </xsl:template>
    
    <xsl:template match="/" mode="fn-matches23">
        <!-- try to match with ^\p{X}(%%%)(:?\p{X}\1){Y}\p{X}$, where Y + 2 is the total number of characters in this range -->
        <xsl:variable name="regex" select="concat('^', $posclass, '(%%%)(:?', $posclass, '\1){', count(*/c) - 2, '}', $posclass, '$')" />
        <res charclass="{$charclass}" regex="{$regex}" count="{count(*/c)}">
            <xsl:value-of select="matches(string-join(*/c/@c, '%%%'), $regex)" />
        </res>
    </xsl:template>
    
    <xsl:template match="/" mode="fn-matches24">
        <!-- try to match with ^\P{X}(%%%)(:?\P{X}\1){Y}\P{X}$, where Y + 2 is the total number of characters outside this range -->
        <xsl:variable name="c" select="*/c/@d/xs:integer(.)" />
        <xsl:variable name="allcharsnot" as="xs:string*">
            <xsl:sequence select="for $i in $validrange[not(. = $c)] return codepoints-to-string($i)"/>                
        </xsl:variable>
        <xsl:variable name="regex" select="concat('^', $negclass, '(%%%)(:?', $negclass, '\1){', count($allcharsnot) - 2, '}', $negclass, '$')" />
        <res charclass="{$charclass}" regex="{$regex}" count="{count($allcharsnot)}">            
            <xsl:value-of select="matches(string-join($allcharsnot, '%%%'), $regex)" />
        </res>
    </xsl:template>
    
    <xsl:template match="/" mode="fn-string-length1">
        <!-- tests whether length of a string with these codepoints is correctly calculated -->
        <xsl:variable name="len" select="string-length(string-join(*/c/@c, ''))" />
        <res string-length="{$len}" char-count="{count(*/c)}">
            <xsl:value-of select="$len = count(*/c) " />
        </res>
    </xsl:template>

    <xsl:template match="/" mode="fn-string-length2">
        <!-- tests whether length of a string outside the range of these codepoints is correctly calculated -->
        <xsl:variable name="c" select="*/c/@d/xs:integer(.)" />
        <xsl:variable name="allcharsnot" as="xs:string*">
            <xsl:sequence select="for $i in $validrange[not(. = $c)] return codepoints-to-string($i)"/>                
        </xsl:variable>
        <xsl:variable name="len" select="string-length(string-join($allcharsnot, ''))" />
        <res string-length="{$len}" char-count="{count($allcharsnot)}">
            <xsl:value-of select="$len = count($allcharsnot) " />
        </res>
    </xsl:template>
        
    <xsl:template match="/" mode="fn-translate1">
        <!-- tests whether we can translate these characters properly -->
        <xsl:variable name="str" select="string-join(*/c/@c, '')" />
        <xsl:variable name="rev" select="string-join(reverse(*/c/@c), '')" />
        <res>
            <xsl:value-of select="translate($str, $str, $rev) = $rev" />
        </res>
    </xsl:template>
            
    <xsl:template match="/" mode="fn-translate2">
        <!-- tests whether we can translate these characters properly -->
        <xsl:variable name="str" select="string-join(*/c/@c, '')" />
        <xsl:variable name="rev" select="string-join(reverse(*/c/@c), '')" />
        <res>
            <xsl:value-of select="translate($str, $rev, $str) = $rev" />
        </res>
    </xsl:template>
        
    <xsl:template match="/" mode="fn-translate3">
        <!-- tests whether we can translate these characters properly into oblivion -->
        <xsl:variable name="str" select="string-join(*/c/@c, '')" />
        <res>
            <xsl:value-of select="translate($str, $str, '') = ''" />
        </res>
    </xsl:template>

    <xsl:template match="/" mode="fn-translate4">
        <!-- tests whether we can translate these characters properly into oblivion using reverse string -->
        <xsl:variable name="str" select="string-join(*/c/@c, '')" />
        <xsl:variable name="rev" select="string-join(reverse(*/c/@c), '')" />
        <res>
            <xsl:value-of select="translate($str, $rev, '') = ''" />
        </res>
    </xsl:template>
    
    <xsl:template match="/" mode="fn-replace1">
        <!-- tests whether each character is replaced using \p{X} -->
        <xsl:variable name="regex" select="$posclass" />
        <res charclass="{$charclass}" regex="{$regex}" replace="**">
            <xsl:value-of select="string-length(string-join(*/c/replace(@c, $regex, '**'), '')) = count(*/c) * 2" />
        </res>
    </xsl:template>
    
    <xsl:template match="/" mode="fn-replace2">
        <!-- tests whether each character is replaced using \P{X}  -->
        <xsl:variable name="regex" select="$negclass" />
        <res charclass="{$charclass}" regex="{$regex}" replace="**">
            <xsl:variable name="c" select="*/c/@d/xs:integer(.)" />
            <xsl:variable name="allcharsnot" as="xs:string*">
                <xsl:sequence select="for $i in $validrange[not(. = $c)] return codepoints-to-string($i)"/>                
            </xsl:variable>
            <xsl:value-of select="string-length(string-join(for $char in $allcharsnot return replace($char, $regex, '**'), '')) = count($allcharsnot) * 2" />
        </res>
    </xsl:template>
        
    <xsl:template match="/" mode="fn-replace3">
        <!-- tests whether each character is replaced with itself (\p{X})(\1)? with $1$2 -->
        <xsl:variable name="regex" select="concat('(', $posclass, ')(\1)?')" />
        <res charclass="{$charclass}" regex="{$regex}" replace="$1$2">
            <xsl:value-of select="string-join(*/c/replace(@c, $regex, '$1$2'), '') = string-join(*/c, '')" />
        </res>
    </xsl:template>
    
    <xsl:template match="/" mode="fn-replace4">
        <!-- tests whether each character is replaced with itself (\P{X})(\1)? with $1$2 -->
        <xsl:variable name="regex" select="concat('(', $negclass, ')(\1)?')" />
        <res charclass="{$charclass}" regex="{$regex}" replace="$1$2">
            <xsl:variable name="c" select="*/c/@d/xs:integer(.)" />
            <xsl:variable name="allcharsnot" as="xs:string*">
                <xsl:sequence select="for $i in $validrange[not(. = $c)] return codepoints-to-string($i)"/>                
            </xsl:variable>
            <xsl:value-of select="string-join(for $char in $allcharsnot return replace($char, $regex, '$1$2'), '') = string-join($allcharsnot, '')" />
        </res>
    </xsl:template>
        
    <xsl:template match="/" mode="fn-replace5">
        <!-- tests whether NOTHING gets replaced with for anying in \p{X}, when regex is (\P{X}) and repl string is '||$1||' -->
        <xsl:variable name="regex" select="concat('(', $negclass, ')')" />
        <res charclass="{$charclass}" regex="{$regex}" replace="||$1||">
            <xsl:value-of select="string-join(*/c/replace(@c, $regex, '||$1||'), '') = string-join(*/c, '')" />
        </res>
    </xsl:template>
    
    <xsl:template match="/" mode="fn-replace6">
        <!-- tests whether NOTHING gets replaced with for anying in \P{X}, when regex is (\p{X}) and repl string is '||$1||' -->
        <xsl:variable name="regex" select="concat('(', $posclass, ')')" />
        <res charclass="{$charclass}" regex="{$regex}" replace="||$1||">
            <xsl:variable name="c" select="*/c/@d/xs:integer(.)" />
            <xsl:variable name="allcharsnot" as="xs:string*">
                <xsl:sequence select="for $i in $validrange[not(. = $c)] return codepoints-to-string($i)"/>                
            </xsl:variable>
            <xsl:value-of select="string-join(for $char in $allcharsnot return replace($char, $regex, '||$1||'), '') = string-join($allcharsnot, '')" />
        </res>
    </xsl:template>
    
    <xsl:template match="/" mode="fn-replace7">
        <!-- take all valid Unicode characters and replace each \p{X} with nothing, result should be same as all \p{X} characters  -->
        <xsl:variable name="regex" select="$posclass" />
        <res charclass="{$charclass}" regex="{$regex}" replace="">
            <xsl:variable name="c" select="*/c/@d/xs:integer(.)" />
            <xsl:variable name="allcharsnotString" as="xs:string">
                <xsl:sequence select="string-join(for $i in $validrange[not(. = $c)] return codepoints-to-string($i), '')"/>                
            </xsl:variable>
            <xsl:variable name="allUnicodeCharsString" as="xs:string*" select="string-join(for $i in $validrange return codepoints-to-string($i), '')" />
            <xsl:value-of select="string-join(replace($allUnicodeCharsString, $regex, ''), '') = $allcharsnotString" />
        </res>
    </xsl:template>
    
    <xsl:template match="/" mode="fn-replace8">
        <!-- take all valid Unicode characters and replace each \P{X} with nothing, result should be same as all \p{X} characters  -->
        <xsl:variable name="regex" select="$negclass" />
        <res charclass="{$charclass}" regex="{$regex}" replace="">
            <xsl:variable name="allUnicodeCharsString" as="xs:string*" select="string-join(for $i in $validrange return codepoints-to-string($i), '')" />
            <xsl:value-of select="string-join(replace($allUnicodeCharsString, $regex, ''), '') = string-join(*/c/@c, '')" />
        </res>
    </xsl:template>
    
</xsl:stylesheet>