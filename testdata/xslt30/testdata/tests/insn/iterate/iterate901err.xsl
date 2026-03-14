<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">

<!-- Test of xsl:iterate; uses a parameter. Error, xsl:param must come first -->

<xsl:output indent="yes"/>

<xsl:template match="/">
<out>
  <xsl:iterate select="//ITEM">
    <xsl:variable name="x" select="29"/>
    <xsl:param name="basketCost" as="xs:decimal" select="$x"/>
    <item cost="{$basketCost}"><xsl:copy-of select="TITLE"/></item>
    <xsl:next-iteration>
      <xsl:with-param name="basketCost" select="$basketCost + (xs:decimal(PRICE), 0)[1]"/>
    </xsl:next-iteration>
  </xsl:iterate>
</out>
</xsl:template>

</xsl:stylesheet>