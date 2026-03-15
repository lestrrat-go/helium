<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<!-- PURPOSE:  Deeply nested dynamic error -->
<?spec fo#func-numeric-integer-divide?><?error FOAR0001?>

<xsl:template match="/">
<xsl:call-template name="doc"/>
</xsl:template>

<xsl:template name="doc">
<xsl:param name="count" select="10"/>
<out>
<xsl:choose>
<xsl:when test="$count gt 0">
  <xsl:call-template name="doc">
    <xsl:with-param name="count" select="$count - 1"/>
  </xsl:call-template>
</xsl:when>
<xsl:otherwise>
  <xsl:value-of select="20 div $count"/>
</xsl:otherwise>
</xsl:choose>
</out>
</xsl:template>
</xsl:stylesheet>