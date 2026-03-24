<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-path-expressions?>
  <!-- Purpose: Tests apply-templates starting with a attribute axes
       followed by additional relative-location-path steps. Note that
       namespace nodes should not be included on the preceeding axis,
       and specifically that the implied declaration of xml: should
       not appear in the output. -->
  <!-- Author: Paul Dick -->

<xsl:template match="/">
  <out>
    <xsl:for-each select="//center">
      <xsl:apply-templates select="@*/ancestor::*/near-north/*[4]/@*/preceding::node()"/>
    </xsl:for-each>
  </out>
</xsl:template>

<xsl:template match="node()">
  <xsl:text>#</xsl:text><xsl:value-of select="name(.)"/>
</xsl:template>

</xsl:stylesheet>
