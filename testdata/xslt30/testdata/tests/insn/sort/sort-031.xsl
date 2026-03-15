<?xml version="1.0" encoding="iso-8859-1" ?>
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


    <!-- Purpose: Reverse the order of the nodes using an expression that is reliable. -->

<xsl:template match="/">
  <out>
  <xsl:for-each select="doc/alpha">
    <xsl:sort select="count(following-sibling::*)" data-type="number"/>
    <xsl:value-of select="@key"/>
  </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>