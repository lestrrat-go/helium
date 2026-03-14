<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    xmlns:f="urn:functions"
    exclude-result-prefixes="xs math f"
    version="3.0">
    
    <xsl:param name="function-name" static="yes" select="'f:wrong'" />
    
    <!-- purpose: testing various naming possibilities for initial functions -->
    
    <xsl:function name="f:square1" visibility="public">
        <xsl:param name="p" as="xs:integer" />
        <xsl:sequence select="$p * $p" />
    </xsl:function>
    
    <xsl:function name="Q{urn:functions}square2" visibility="public">
        <xsl:param name="p" as="xs:integer" />
        <xsl:sequence select="$p * $p" />
    </xsl:function>
    
    <xsl:function _name="f:square3" visibility="public">
        <xsl:param name="p" as="xs:integer" />
        <xsl:sequence select="$p * $p" />
    </xsl:function>
    
    <xsl:function _name="{'f:'}{'square' || 4}" visibility="public">
        <xsl:param name="p" as="xs:integer" />
        <xsl:sequence select="$p * $p" />
    </xsl:function>
    
    <xsl:function _name="{QName('urn:functions', 'x:square5')}" xmlns:x="urn:functions" visibility="public">
        <xsl:param name="p" as="xs:integer" />
        <xsl:sequence select="$p * $p" />
    </xsl:function>
    
    <xsl:function _name="{$function-name}" visibility="public">
        <xsl:param name="p" as="xs:integer" />
        <xsl:sequence select="$p * $p" />
    </xsl:function>
    
</xsl:stylesheet>