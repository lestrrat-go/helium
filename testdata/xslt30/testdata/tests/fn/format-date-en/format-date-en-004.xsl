<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec xslt#format-date?>
  <!-- PURPOSE: XSLT 2.0: test format-date: English month names, upper case, three chars -->

  <xsl:param name="d" as="xs:date" select="xs:date('2003-12-07')"/>

  <xsl:template name="main">
<out>;
 <xsl:for-each select="1 to 12">
   <xsl:variable name="d2" select="$d + xs:yearMonthDuration('P1M')*."/>
   <z><xsl:value-of select="format-date($d2, '[MN,3-3]', 'en', (), ())"/></z>;
 </xsl:for-each>  
</out>     
  </xsl:template>

</xsl:stylesheet>