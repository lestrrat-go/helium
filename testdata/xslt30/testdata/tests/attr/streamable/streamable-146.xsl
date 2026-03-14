<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    exclude-result-prefixes="xs math" 
    version="3.0">
    
    <xsl:mode streamable="yes"/>
    
    <!-- Saxon bug 3465: we were loop-lifting the expression @* which made the code non-streamable -->
    
    <xsl:variable name="columns" as="xs:string*" select="tokenize('id,dummyAttr1,dummyAttr2,dummyAttr3,dummyAttr4', ',')"/>
    
    <xsl:template match="/">
        <out>
            <header><xsl:value-of select="$columns" separator=","/></header>
            <xsl:apply-templates select="outermost(//itemA)"/>
        </out>
    </xsl:template>
    
    <xsl:template match="itemA">
        <row><xsl:value-of select="for $col in $columns return @*[name() eq $col] || ''" separator=","/></row>
    </xsl:template>
    
</xsl:stylesheet>
