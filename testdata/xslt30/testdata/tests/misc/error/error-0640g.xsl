<?xml version="1.0" encoding="UTF-8"?>
<!--In general, a circularity in a stylesheet is a non-recoverable dynamic error.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test use of key() in a key use expression (XSLT 2.0) with circular definitions -->
<?spec xslt#errors?><?error XPDE0640?>

  <xsl:key name="k1" match="a" use="key('k2',@code)"/>
  <xsl:key name="k2" match="a" use="key('k1',@code)"/>
  <xsl:param name="p" select="'2'"/>

  <xsl:template match="/">
      <out total-one="{sum(key('k1', '2')/@value)}"
           total-two="{sum(key('k2', '1')/@value)}"/>
  </xsl:template>
   
</xsl:stylesheet>
