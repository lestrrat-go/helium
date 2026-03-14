<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:map="http://www.w3.org/2005/xpath-functions/map" 
    xmlns:mf="http://example.com/mf" 
    exclude-result-prefixes="xs map mf" 
    version="3.0">    
    
    <xsl:function name="mf:f1">
        <xsl:sequence select="map { 'value' : 0 }"/>
    </xsl:function>
    
    <xsl:template match="/" name="xsl:initial-template">
        <xsl:value-of select="mf:f1()"/>
    </xsl:template>
    
</xsl:stylesheet>