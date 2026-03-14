<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    version="3.0" expand-text="yes">

    <!-- 
        Rudimentary but excessive test for deeper dotted relative paths with resolve-uri and base uris
        Test covers:
        - dot elements in relative base uri
        - dot elements in rel uri
        - dot elements that try to get past the auth segment (should return scheme '://' auth '/')
        - absolute path in base uri that, because it must be resolved, acc to RFC 3986, should remove dot segments, quote:
        - none of these tests have $relative ending with a non-dotted segment, which means that all resolution will end with trailing "/"
        
        "However, some deployed implementations incorrectly assume that reference resolution
        is not necessary when the reference is already a URI and thus fail to remove 
        dot-segments when they occur in non-relative paths.  URI normalizers should remove 
        dot-segments by applying the remove_dot_segments algorithm to the path, as described 
        in Section 5.2.4."
    -->
    

    <xsl:output indent="no"></xsl:output>

    <xsl:template name="initial-template">
        
        <out xml:base="http://example.com/a/b/c/d/../d/e/f/g/i/j/k/l/m/n/o/p/q/r/s/t/u/v/w/x/y/z">
            <xsl:variable name="base" select="static-base-uri()" xml:base="Z/../Z"/>
            
            <!-- varying bases -->
            <base1>{$base}</base1>
            <base2>{static-base-uri()}</base2>
            <base3>{resolve-uri(static-base-uri(), static-base-uri())}</base3>
            
            <!-- ignorable dots, *but* removes the "z", leaves the trailing "/" -->
            <resolveA>{resolve-uri('.')}</resolveA>
            <resolveB>{resolve-uri('./')}</resolveB>
            <resolveC>{resolve-uri('./././././././')}</resolveC>
            
            <!-- variants of trying to get past the root, should return root path acc. to RFC, and *must* include root path leading "/" -->
            <resolveD>{resolve-uri('/')}</resolveD>
            <resolveE>{resolve-uri('/./././././././')}</resolveE>
            <resolveF>{resolve-uri('/../../.././././././././')}</resolveF>
            <resolveG>{resolve-uri('/..')}</resolveG>
            <resolveH>{resolve-uri('/../')}</resolveH>
            <resolveI>{resolve-uri('../', 'http://example.com')}</resolveI>
            <resolveJ>{resolve-uri('../Z/..', 'http://example.com')}</resolveJ>
            <resolveK>{resolve-uri('./.', 'http://example.com')}</resolveK>
            
            <!-- variants of one level up, ends with "/x/" -->
            <resolveL>{resolve-uri('..')}</resolveL>
            <resolveM>{resolve-uri('../')}</resolveM>
            <resolveN>{resolve-uri('../.')}</resolveN>
            <resolveO>{resolve-uri('.././')}</resolveO>
            <resolveP>{resolve-uri('.././')}</resolveP>
            <resolveQ>{resolve-uri('./././.././.')}</resolveQ>
            
            <!-- tests overlap of dotted path, where multiple dots on the right of a non-dotted segment could (with a wrong algo) potentially remove dotted elements further to the left -->
            <resolveR>{resolve-uri('./X/Y/Z/../W/../../.././..')}</resolveR>
            <resolveS>{resolve-uri('../../.././X/Y/Z/../W/../../.././../v/w/x/')}</resolveS>
            
            <!-- each following line ends with a lower letter in the alphabet, starting with "/w/" -->
            <resolveT>{resolve-uri('../../')}</resolveT>
            <resolveU>{resolve-uri('../../../')}</resolveU>
            <resolveV>{resolve-uri('../../../../')}</resolveV>
            <resolveW>{resolve-uri('../../../../../')}</resolveW>
            <resolveX>{resolve-uri('../../../../../../')}</resolveX>
            <resolveY>{resolve-uri('../../../../../../../')}</resolveY>
            <resolveZ>{resolve-uri('../../../../../../../../')}</resolveZ>
            <resolveZA>{resolve-uri('../../../../../../../../../')}</resolveZA>
            <resolveZB>{resolve-uri('../../../../../../../../../../')}</resolveZB>
            <resolveZC>{resolve-uri('../../../../../../../../../../../')}</resolveZC>
            <resolveZD>{resolve-uri('../../../../../../../../../../../../')}</resolveZD>
            <resolveZE>{resolve-uri('../../../../../../../../../../../../../')}</resolveZE>
            <resolveZF>{resolve-uri('../../../../../../../../../../../../../../')}</resolveZF>
            <resolveZG>{resolve-uri('../../../../../../../../../../../../../../../')}</resolveZG>
            <resolveZH>{resolve-uri('../../../../../../../../../../../../../../../../')}</resolveZH>
            <resolveZI>{resolve-uri('../../../../../../../../../../../../../../../../../')}</resolveZI>
            <resolveZJ>{resolve-uri('../../../../../../../../../../../../../../../../../../')}</resolveZJ>
            <resolveZK>{resolve-uri('../../../../../../../../../../../../../../../../../../../')}</resolveZK>
            <resolveZL>{resolve-uri('../../../../../../../../../../../../../../../../../../../../')}</resolveZL>
            <resolveZM>{resolve-uri('../../../../../../../../../../../../../../../../../../../../../')}</resolveZM>
            <resolveZN>{resolve-uri('../../../../../../../../../../../../../../../../../../../../../../')}</resolveZN>
            <resolveZO>{resolve-uri('../../../../../../../../../../../../../../../../../../../../../../../')}</resolveZO>
            
            <!-- variants that should end with "/a/" -->
            <resolveZP>{resolve-uri('./././../../../../../../../../../././././../../../../../../../../../../../../../../././.')}</resolveZP>
            
            <!-- variants that try to get past the root, should return root with dotted parts removed -->
            <resolveZQ>{resolve-uri('../../../../../../../../../../../../../../../../../../../../../../../../')}</resolveZQ>
            <resolveZR>{resolve-uri('../../../../../../../../../../../../../../../../../../../../../../../../../')}</resolveZR>
            <resolveZS>{resolve-uri('../../../../../../../../../../../../../../../../../../../../../../../../../../')}</resolveZS>
            <resolveZT>{resolve-uri('/../../../../../../../../../../../../../../../../../../../../../../../')}</resolveZT>
        </out>
    </xsl:template>

</xsl:stylesheet>
