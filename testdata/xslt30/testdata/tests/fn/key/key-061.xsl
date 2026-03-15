<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xslt#key?>
<!-- Test use of variables in a key match pattern (XSLT 2.0) -->

  <xsl:key name="k" match="a[@code=$p]" use="@code"/>
  <xsl:param name="p" select="'2'"/>

  <xsl:template match="/">
    <out total="{sum(key('k', '2')/@value)}"/>
  </xsl:template>
   
</xsl:stylesheet>
