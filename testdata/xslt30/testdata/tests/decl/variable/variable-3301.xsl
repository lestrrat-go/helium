<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    version="2.0">

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<?spec xslt#local-variables?>
  <!-- Purpose: Show that one param (p2) can be set to value of another (p1)
    equally well in apply and call template invocations. -->
  

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="*"/>
    <xsl:text>&#10;== apply above, call below ==&#10;</xsl:text>
    <xsl:for-each select="//*">
      <xsl:call-template name="matchElem"/>
    </xsl:for-each>
  </out>
</xsl:template>

<xsl:template match="*">
  <xsl:param name="p1" select="@id"/>
  <xsl:param name="p2" select="$p1"/>
  <xsl:value-of select="$p1"/>,  <xsl:value-of select="$p2"/>
  <xsl:text>&#10;</xsl:text>
  <xsl:apply-templates select="*"/><!-- Recursively visit all children -->
</xsl:template>

<xsl:template name="matchElem">
  <xsl:param name="p1" select="@id"/>
  <xsl:param name="p2" select="$p1"/>
  <xsl:value-of select="$p1"/>,  <xsl:value-of select="$p2"/>
  <xsl:text>&#10;</xsl:text>
</xsl:template>

</xsl:stylesheet>