<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec xslt#format-date?>
  <!-- PURPOSE: XSLT 2.0: test format-date: English day names, title case, variable chars -->

  <xsl:param name="d" as="xs:date" select="xs:date('2003-12-07')"/>
  
  <xsl:variable name="expected" select="'Mon', 'Tues', 'Weds', 'Thurs', 'Fri', 'Sat', 'Sun'"/>  

  <xsl:template name="main">
		<out>
		 <xsl:for-each select="1 to 7">
		   <xsl:variable name="n" select="."/>
		   <xsl:variable name="d2" select="$d + xs:dayTimeDuration('P1D')*."/>
		   <xsl:variable name="abb" select="format-date($d2, '[FNn,3-5]', 'en', (), ())"/>
		   <z><xsl:value-of select="substring($abb, 1, 3), 
		                            starts-with($expected[$n], $abb) and string-length($abb) le 5 and string-length($abb) ge 3"/></z>
		 </xsl:for-each>  
		</out>     
  </xsl:template>

</xsl:stylesheet>