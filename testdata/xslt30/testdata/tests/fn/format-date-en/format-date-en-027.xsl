<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec xslt#format-date?>
  <!-- PURPOSE: XSLT 2.0: test format-date: English ordinal words 1990-2020 lower case-->

  <xsl:param name="d" as="xs:date" select="xs:date('1990-12-01')"/>
  <xsl:param name="debug" select="'no'"/>
  <xsl:template name="main">
<out>
 <xsl:variable name="out">
 <xsl:for-each select="0 to 30">
   <xsl:variable name="d2" select="$d + xs:yearMonthDuration('P1Y')*."/>
   <z><xsl:value-of select="format-date($d2, '[Ywo]', 'en', (), ())"/></z>;
 </xsl:for-each> 
 </xsl:variable>
 <xsl:if test="$debug='yes'"><xsl:copy-of select="$out"/></xsl:if>
 <xsl:value-of select="every $x in $out/z satisfies contains($x, 'thousand')"/> 
</out>     
  </xsl:template>

</xsl:stylesheet>