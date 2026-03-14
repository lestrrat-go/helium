<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
    <!-- Purpose: Test id() in a xsl:number from pattern, with a local variable. -->


<xsl:template match="/">
  <xsl:param name="a" select="'  a  d '"/>
  <out>
    <xsl:for-each select="tee/s/r">
      <g><xsl:number from="id($a)" level="any"/></g>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
