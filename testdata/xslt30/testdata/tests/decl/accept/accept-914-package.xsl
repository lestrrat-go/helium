<xsl:package
    name="http://www.w3.org/xslt30tests/package-914"  
    package-version="1.0.0"
    version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:a="urn:attribute-set">
    
    <!-- tests error XTSE3080 (not XTDE0040 or XTDE3052) to be raised on abstract initial template, causing stylesheet to be non-executable -->

    <xsl:template name="xsl:initial-template" visibility="abstract">
    </xsl:template>
    
</xsl:package>   