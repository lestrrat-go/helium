<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <?spec fo#casting-to-numerics?>
<!-- PURPOSE:  a recursive decimal-to-hex conversion -->

<xsl:variable name="hexDigits" select="'0123456789ABCDEF'" />

<xsl:template match="/">
  <o><xsl:apply-templates/></o>
</xsl:template>

<xsl:template name="toHex">
    <xsl:param name="decimalNumber" />
    <xsl:if test="$decimalNumber >= 16">
      <xsl:call-template name="toHex">
        <xsl:with-param name="decimalNumber" select="floor($decimalNumber div 16)" />
      </xsl:call-template>
    </xsl:if>
    <xsl:value-of select="substring($hexDigits, ($decimalNumber mod 16) + 1, 1)" />
</xsl:template>

<xsl:template match="in">
  <out dec="{.}">
    <xsl:call-template name="toHex">
      <xsl:with-param name="decimalNumber" select="number(.)"/>
    </xsl:call-template>   
  </out>
</xsl:template>
 
</xsl:stylesheet>
