<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#key?>
<!-- Test use of variables in a key use expression (XSLT 2.0) -->

  <xsl:key name="k" match="a" use="@code[.=$p]"/>
  <xsl:param name="p" select="'2'"/>

  <xsl:template match="/">
    <out total-two="{sum(key('k', '2')/@value)}" total-one="{sum(key('k', '1')/@value)}"/>
  </xsl:template>
   
</xsl:stylesheet>
