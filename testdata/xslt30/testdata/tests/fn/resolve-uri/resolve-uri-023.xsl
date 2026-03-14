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
    
    <!-- Copy of resolve-uri-022 changed to require run-time resolution -->
    

    <xsl:output indent="no"></xsl:output>
    
    <xsl:param name="p" select="''"/>

    <xsl:template name="initial-template">
        
        <out xml:base="http://example.com/a/b/c/d/../d/e/f/g/i/j/k/l/m/n/o/p/q/r/s/t/u/v/w/x/y/z">
            <xsl:variable name="base" select="static-base-uri()" xml:base="Z/../Z"/>
            
            <!-- ignorable dots, *but* removes the "z", leaves the trailing "/" -->
            <resolveA>{resolve-uri($p||'.')}</resolveA>
            <resolveB>{resolve-uri($p||'./')}</resolveB>
            <resolveC>{resolve-uri($p||'./././././././')}</resolveC>
            
            <!-- variants of trying to get past the root, should return root path acc. to RFC, and *must* include root path leading "/" -->
            <resolveD>{resolve-uri($p||'/')}</resolveD>
            <resolveE>{resolve-uri($p||'/./././././././')}</resolveE>
            <resolveF>{resolve-uri($p||'/../../.././././././././')}</resolveF>
            <resolveG>{resolve-uri($p||'/..')}</resolveG>
            <resolveH>{resolve-uri($p||'/../')}</resolveH>
            <resolveI>{resolve-uri($p||'../', 'http://example.com')}</resolveI>
            <resolveJ>{resolve-uri($p||'../Z/..', 'http://example.com')}</resolveJ>
            <resolveK>{resolve-uri($p||'./.', 'http://example.com')}</resolveK>
            
            <!-- variants of one level up, ends with "/x/" -->
            <resolveL>{resolve-uri($p||'..')}</resolveL>
            <resolveM>{resolve-uri($p||'../')}</resolveM>
            <resolveN>{resolve-uri($p||'../.')}</resolveN>
            <resolveO>{resolve-uri($p||'.././')}</resolveO>
            <resolveP>{resolve-uri($p||'.././')}</resolveP>
            <resolveQ>{resolve-uri($p||'./././.././.')}</resolveQ>
            
            <!-- tests overlap of dotted path, where multiple dots on the right of a non-dotted segment could (with a wrong algo) potentially remove dotted elements further to the left -->
            <resolveR>{resolve-uri($p||'./X/Y/Z/../W/../../.././..')}</resolveR>
            <resolveS>{resolve-uri($p||'../../.././X/Y/Z/../W/../../.././../v/w/x/')}</resolveS>
            
            <!-- each following line ends with a lower letter in the alphabet, starting with "/w/" -->
            <resolveT>{resolve-uri($p||'../../')}</resolveT>
            <resolveU>{resolve-uri($p||'../../../')}</resolveU>
            <resolveV>{resolve-uri($p||'../../../../')}</resolveV>
            <resolveW>{resolve-uri($p||'../../../../../')}</resolveW>
            <resolveX>{resolve-uri($p||'../../../../../../')}</resolveX>
            <resolveY>{resolve-uri($p||'../../../../../../../')}</resolveY>
            <resolveZ>{resolve-uri($p||'../../../../../../../../')}</resolveZ>
            <resolveZA>{resolve-uri($p||'../../../../../../../../../')}</resolveZA>
            <resolveZB>{resolve-uri($p||'../../../../../../../../../../')}</resolveZB>
            <resolveZC>{resolve-uri($p||'../../../../../../../../../../../')}</resolveZC>
            <resolveZD>{resolve-uri($p||'../../../../../../../../../../../../')}</resolveZD>
            <resolveZE>{resolve-uri($p||'../../../../../../../../../../../../../')}</resolveZE>
            <resolveZF>{resolve-uri($p||'../../../../../../../../../../../../../../')}</resolveZF>
            <resolveZG>{resolve-uri($p||'../../../../../../../../../../../../../../../')}</resolveZG>
            <resolveZH>{resolve-uri($p||'../../../../../../../../../../../../../../../../')}</resolveZH>
            <resolveZI>{resolve-uri($p||'../../../../../../../../../../../../../../../../../')}</resolveZI>
            <resolveZJ>{resolve-uri($p||'../../../../../../../../../../../../../../../../../../')}</resolveZJ>
            <resolveZK>{resolve-uri($p||'../../../../../../../../../../../../../../../../../../../')}</resolveZK>
            <resolveZL>{resolve-uri($p||'../../../../../../../../../../../../../../../../../../../../')}</resolveZL>
            <resolveZM>{resolve-uri($p||'../../../../../../../../../../../../../../../../../../../../../')}</resolveZM>
            <resolveZN>{resolve-uri($p||'../../../../../../../../../../../../../../../../../../../../../../')}</resolveZN>
            <resolveZO>{resolve-uri($p||'../../../../../../../../../../../../../../../../../../../../../../../')}</resolveZO>
            
            <!-- variants that should end with "/a/" -->
            <resolveZP>{resolve-uri($p||'./././../../../../../../../../../././././../../../../../../../../../../../../../../././.')}</resolveZP>
            
            <!-- variants that try to get past the root, should return root with dotted parts removed -->
            <resolveZQ>{resolve-uri($p||'../../../../../../../../../../../../../../../../../../../../../../../../')}</resolveZQ>
            <resolveZR>{resolve-uri($p||'../../../../../../../../../../../../../../../../../../../../../../../../../')}</resolveZR>
            <resolveZS>{resolve-uri($p||'../../../../../../../../../../../../../../../../../../../../../../../../../../')}</resolveZS>
            <resolveZT>{resolve-uri($p||'/../../../../../../../../../../../../../../../../../../../../../../../')}</resolveZT>
        </out>
    </xsl:template>

</xsl:stylesheet>
