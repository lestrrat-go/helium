<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    expand-text="yes"
    version="3.0">
    
    <xsl:variable name="Hello">Hello</xsl:variable>
    
    <xsl:include href="cvt-048inc.xsl"/>
    
    <xsl:variable name="World">World</xsl:variable>
    
    <xsl:template name="xsl:initial-template">
        <xsl:call-template name="sub"/>
    </xsl:template>
    
</xsl:stylesheet>