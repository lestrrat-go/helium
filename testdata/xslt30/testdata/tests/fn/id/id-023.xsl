<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
    <!-- Purpose: Test id() in a key match pattern. -->

<xsl:param name="a" select="'  a  d '"/>
<xsl:key name="k" match="id($a)" use="r/@size"/>
<xsl:template match="/">
  <out>
    <x><xsl:value-of select="key('k', '20')/@id"/></x>
    <y><xsl:value-of select="key('k', '18')/@id"/></y>
  </out>
</xsl:template>

</xsl:stylesheet>
