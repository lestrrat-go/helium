<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#keys?>
<!-- Test use of key() in a key match expression (XSLT 2.0) -->

  <xsl:key name="k1" match="key('k2', '2')" use="@code"/>
  <xsl:key name="k2" match="a" use="@code"/>
  <xsl:param name="p" select="'2'"/>

  <xsl:template match="/">
    <out total-one="{sum(key('k1', '2')/@value)}" total-two="{sum(key('k2', '1')/@value)}"/>
  </xsl:template>
   
</xsl:stylesheet>
