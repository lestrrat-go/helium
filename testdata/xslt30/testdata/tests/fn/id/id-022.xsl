<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
    <!-- Purpose: Test id() in a for-each-group ending-with pattern, with a local variable. -->


<xsl:template match="/">
  <xsl:param name="a" select="'  a  d '"/>
  <out>
    <xsl:for-each-group select="tee/s/r" group-ending-with="id($a)/r" >
      <g><xsl:value-of select="current-group()/@size"/></g>
    </xsl:for-each-group>
  </out>
</xsl:template>

</xsl:stylesheet>
