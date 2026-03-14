<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
   version="2.0">
    <!-- Purpose: Saxon bug 3722. xsl:param on template containing forwards and backwards references to global variables -->

    <xsl:variable name="global1" select="20"/>

    <xsl:template name="main">
        <out>
            <xsl:call-template name="subsid"/>
        </out>
    </xsl:template>
    
    <xsl:template name="subsid">
        <xsl:param name="p" select="$global1 + $global2" as="xs:integer"/>
        <in value="{$p}"/>
    </xsl:template>
    
    <xsl:variable name="global2" select="22"/>
    
    
</xsl:transform>
