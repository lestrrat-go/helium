<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- test key() in a template match pattern, with a variable reference, 
     not the last step in the pattern -->

  <xsl:key name="k1" match="*" use="Name/@First"/>
  <xsl:param name="name">Ted</xsl:param>

  <xsl:template match="/">
    <out>
      <xsl:apply-templates select="//Name"/>
    </out>
  </xsl:template>

  <xsl:template match="key('k1', $name)//Name">
    <desc of="{$name}"><xsl:value-of select="@First"/></desc>
  </xsl:template>

  <xsl:template match="Name">
    <non-desc of="{$name}"><xsl:value-of select="@First"/></non-desc>
  </xsl:template>

</xsl:stylesheet>
