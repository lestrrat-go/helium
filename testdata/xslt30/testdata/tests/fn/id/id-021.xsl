<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
    <!-- Purpose: Test id() in a for-each-group starting-with pattern, with variables. -->

<xsl:param name="a" select="'  a  '"/>
<xsl:param name="d" select="'  d  '"/>
<xsl:template match="/">
  <out>
    <xsl:for-each-group select="tee/s" group-starting-with="id($a)|id($d)" >
      <g><xsl:value-of select="current-group()/r/@size"/></g>
    </xsl:for-each-group>
  </out>
</xsl:template>

</xsl:stylesheet>
