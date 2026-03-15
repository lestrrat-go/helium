<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    exclude-result-prefixes="xs math"
    version="3.0">
    
    <xsl:param name="regex" static="true" select="'\p{IsBhaiksuki}'" />
    
    <xsl:template name="xsl:initial-template">        
        <!-- 55296 == U+D800, 57343 == U+DFFF, in other words: surrogate range -->
        <!-- 65534 == U+FFFE, 65536 == U+FFFF, non-characters 
        Adlam: 125184 - 125279
        Bhaiksuki: 72704 - 72815
        Marchen: 72816 - 72895
        Tangut: 94208 - 100351
        -->
        
        <count>
            <xsl:value-of select="count(
                ((70001 to 74000, 125001 to 127000, 93001 to 102000) (: 15000 :)
                ! codepoints-to-string(.)) 
                [matches(., $regex)])
                
                " separator="" />
        </count>
    </xsl:template>
    
</xsl:stylesheet>