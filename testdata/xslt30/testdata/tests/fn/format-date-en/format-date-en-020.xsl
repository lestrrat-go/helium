<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec xslt#format-date?>
  <!-- PURPOSE: XSLT 2.0: test format-date: English cardinal words 1990-2020 upper case-->

  <xsl:param name="d" as="xs:date" select="xs:date('1990-12-01')"/>

  <xsl:template name="main">
<out>;
 <xsl:for-each select="0 to 30">
   <xsl:variable name="d2" select="$d + xs:yearMonthDuration('P1Y')*."/>
   <z><xsl:value-of select="format-date($d2, '[YW]', 'en', (), ())"/></z>;
 </xsl:for-each>  
</out>     
  </xsl:template>

</xsl:stylesheet>