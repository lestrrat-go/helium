<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    xmlns:j="http://www.w3.org/2013/XSLT/xml-to-json" 
    exclude-result-prefixes="#all" 
    version="3.0">
    
    <xsl:use-package name="http://www.w3.org/2013/XSLT/xml-to-json"/>
    
    <xsl:mode streamable="true"/>
    
    <xsl:template match="/">
        <out><xsl:sequence select="j:xml-to-json(.)"/></out>
    </xsl:template>
    
    <xsl:template match="/" mode="local-indent">
        <out><xsl:sequence select="j:xml-to-json(., map{'indent':true()})"/></out>
    </xsl:template>
    
</xsl:stylesheet>