<?xml version="1.0" encoding="UTF-8"?>
<!--In general, a circularity in a stylesheet is a non-recoverable dynamic error.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test use of variables in a key use expression (XSLT 2.0) causing a circularity -->
<?spec xslt#errors?><?error?>

  <xsl:key name="k" match="a" use="@code[.=$p]"/>
  <xsl:param name="p" select="key('k', '2')"/>

  <xsl:template match="/">
      <out total="{sum(key('k', '2')/@value)}"/>
  </xsl:template>

</xsl:stylesheet>
