<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#axes?>
  <!-- Purpose: Tests apply-templates from an attribute using parent axis. -->
  <!-- This is a twist on axes96, where the context node is an element. -->
  
<xsl:template match="/">
  <out>
    <xsl:for-each select="//center/@center-attr-1">
      <xsl:apply-templates select="../.."/>
    </xsl:for-each>
  </out>
</xsl:template>

<xsl:template match="*">
  <xsl:value-of select="name(.)"/>
</xsl:template>

</xsl:stylesheet>