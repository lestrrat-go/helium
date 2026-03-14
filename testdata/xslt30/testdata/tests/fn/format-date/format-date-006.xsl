<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec xslt#format-date?>
  <!-- PURPOSE: XSLT 2.0: test format-date: lower-case roman numerals for year (width specifier causes padding to 4 chars)-->

  <xsl:param name="t" as="xs:date" select="xs:date('0800-01-01')"/>  

  <xsl:template name="main">
<out>;
<xsl:for-each select="1 to 100">
<t><xsl:value-of select="format-date($t + xs:yearMonthDuration('P17Y')*., '[Yi,4-4]')"/></t>;
</xsl:for-each>
</out>     
  </xsl:template>

</xsl:stylesheet>