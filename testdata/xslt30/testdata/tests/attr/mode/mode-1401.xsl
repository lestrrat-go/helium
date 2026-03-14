<?xml version="1.0"?>
<?spec xslt#modes?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:f="http://example.com/modes35" exclude-result-prefixes="f">

<!-- Test that mode="#current" in a function reverts to the unnamed mode (XSLT 2.0, clarfied by erratum XT.E19) -->

<xsl:template match="book">
  <out>
    <xsl:apply-templates select="chapter" mode="m"/>
  </out>
</xsl:template>

<xsl:template match="chapter" mode="m">
  <chapter>
    <xsl:apply-templates select="chtitle" mode="#current"/>
    <xsl:sequence select="f:apply(.)"/>
  </chapter>
</xsl:template>

<xsl:template match="chtitle" mode="m">
  <chtitle mode="m"><xsl:value-of select="."/></chtitle>
</xsl:template>

<xsl:template match="chtitle" mode="#default">
  <chtitle mode="#default"><xsl:value-of select="."/></chtitle>
</xsl:template>

<xsl:function name="f:apply">
  <xsl:param name="this" as="element(chapter)"/>
  <xsl:apply-templates select="$this/chtitle" mode="#current"/>
</xsl:function>


</xsl:stylesheet>
