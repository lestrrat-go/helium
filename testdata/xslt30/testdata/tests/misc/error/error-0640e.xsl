<?xml version="1.0" encoding="UTF-8"?>
<!--In general, a circularity in a stylesheet is a non-recoverable dynamic error.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test use of variables in a key match pattern (XSLT 2.0) causing a circularity -->
<!-- Result disputed for XSLT 3.0: see bug 26543 -->
<?spec xslt#errors?><?error XTDE0640?>

  <xsl:key name="k" match="a[@code=$p]" use="@code"/>
  <xsl:param name="p" select="key('k', '2')"/>

  <xsl:template match="/">
      <!-- in 2.0, the key match may recover by not matching anything, resulting in 0 -->
      <out total="{sum(key('k', '2')/@value)}"/>
  </xsl:template>
   
</xsl:stylesheet>
