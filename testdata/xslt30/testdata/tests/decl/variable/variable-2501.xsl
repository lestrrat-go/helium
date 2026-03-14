<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <?spec xslt#variables?>
  <!-- Section: 11.3 Using Variables and Parameters with xsl:copy-of -->
  <!-- Purpose: Build an RTF from instructions, then use xsl:copy-of. -->
  

<xsl:template match="doc">
  <xsl:variable name="tree">
    <xsl:element name="main">
      <xsl:element name="first">
        <xsl:attribute name="type">text</xsl:attribute>
        <xsl:text>junk</xsl:text>
      </xsl:element>
      <xsl:element name="second">
        <xsl:attribute name="type">fetched</xsl:attribute>
        <xsl:value-of select="." />
      </xsl:element>
      <xsl:element name="third">
        <xsl:attribute name="type">comment</xsl:attribute>
        <xsl:comment>remarks</xsl:comment>
      </xsl:element>
    </xsl:element>
  </xsl:variable>
  <out>
    <xsl:copy-of select="$tree"/>
  </out>
</xsl:template>

</xsl:stylesheet>
