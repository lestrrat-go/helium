<?xml version="1.0" encoding="iso-8859-1" ?>
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Test for xsl:sort using a local variable in the select, apply-templates. -->

  <xsl:template match="/">
    <out>
      <xsl:variable name="index" select="1"/>
      <xsl:apply-templates select="doc/alpha">
        <xsl:sort select="*[$index]" data-type="text"/>
      </xsl:apply-templates>
    </out>
  </xsl:template>

  <xsl:template match="doc/alpha">
    <xsl:value-of select="display"/>
    <xsl:text>,</xsl:text>
  </xsl:template>

</xsl:stylesheet>
