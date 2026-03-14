<?xml version="1.0"?> 
<?spec xslt#key?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test use of key() in a key use expression (XSLT 2.0) -->

  <xsl:key name="k1" match="a" use="@code"/>
  <xsl:key name="k2" match="a" use="key('k1',@code)"/>
  <xsl:param name="p" select="'2'"/>

  <xsl:template match="/">
    <out total-one="{sum(key('k1', '2')/@value)}" total-two="{sum(key('k2', '1')/@value)}"/>
  </xsl:template>
   
</xsl:stylesheet>
