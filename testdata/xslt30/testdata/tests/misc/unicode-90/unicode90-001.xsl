<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    exclude-result-prefixes="xs math"
    version="3.0">
    
    <xsl:param name="regex" static="true" select="'\w'" />
    
    <xsl:template name="xsl:initial-template" match="/">        
        <!-- 
            55296 == U+D800, 57343 == U+DFFF, in other words: surrogate range 
            65534 == U+FFFE, 65536 == U+FFFF, non-characters
            9 = tab, 10 = lf, 13 = cr
         -->
        
        <count>
            <xsl:value-of select="count(
                ((9, 10, 13, 32 to 55295, 57344 to 65533, 65537 to 1114111) (: total 63454 chars :)
                ! codepoints-to-string(.)) 
                [matches(., $regex)])
                
                " separator="" />
        </count>
    </xsl:template>
    
</xsl:stylesheet>