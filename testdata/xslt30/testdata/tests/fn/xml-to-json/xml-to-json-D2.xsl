<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:j="http://www.w3.org/2005/xpath-functions"
    exclude-result-prefixes="xs j"
    version="3.0">
    
    
    <xsl:output method="xml" indent="no"/>
    
    <!-- Valid (but edge case) inputs to xml-to-json -->
    

    
    <xsl:template name="t015">
        <!-- backspace -->
        <xsl:variable name="in"><j:string escaped="1">-<xsl:value-of select="codepoints-to-string(8)"/>-</j:string></xsl:variable>
        <t><xsl:sequence select="xml-to-json($in)"/></t>
    </xsl:template>
    
    <xsl:template name="t017">
        <!-- bell -->
        <xsl:variable name="in"><j:string escaped="1">-<xsl:value-of select="codepoints-to-string(7)"/>-</j:string></xsl:variable>
        <t><xsl:sequence select="xml-to-json($in)"/></t>
    </xsl:template>
    
    <xsl:template name="t018">
        <!-- form feed -->
        <xsl:variable name="in"><j:string escaped="1">-<xsl:value-of select="codepoints-to-string(12)"/>-</j:string></xsl:variable>
        <t><xsl:sequence select="xml-to-json($in)"/></t>
    </xsl:template>
    
  
    
</xsl:stylesheet>