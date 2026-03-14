<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0" default-mode="a">
    
    <!-- PURPOSE:  Test attribute default-mode on included stylesheet. -->
    
    <xsl:template match="a">
        <xsl:text>mode-a:</xsl:text>
        <xsl:value-of select="."/>
    </xsl:template>
    
</xsl:stylesheet>
