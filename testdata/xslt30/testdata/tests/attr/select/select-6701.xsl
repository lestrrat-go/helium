<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#current-function?>

    <!-- Purpose: Simple test of current() in apply-templates. -->

<xsl:template match="doc">
  <out>
    <xsl:apply-templates select="foo">
      <xsl:with-param name="node" select="current()"/><!-- the doc node -->
    </xsl:apply-templates>
  </out>
</xsl:template>

<xsl:template match="foo">
  <xsl:param name="node" select="'duh'"/>
  <xsl:text>&#10;</xsl:text>
  <content>
    <xsl:attribute name="from">
      <xsl:value-of select="@name"/>
    </xsl:attribute>
    <xsl:attribute name="size">
      <xsl:value-of select="count($node)"/>
    </xsl:attribute>
    <xsl:value-of select="normalize-space($node)"/><!-- stringification of doc node -->
  </content>
</xsl:template>

</xsl:stylesheet>
