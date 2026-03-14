<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test backreferences: \10 where there are 10 capturing subgroups -->
  <?spec fo#func-replace?>
  
  <xsl:output encoding="iso-8859-1"/>
  
  <xsl:template match="/">
     <xsl:variable name="t">He said, "I don't" eat abcdefghi'grass'</xsl:variable>
     <xsl:variable name="p">(a)(b)(c)(d)(e)(f)(g)(h)(i)(['"])(.*?)\10</xsl:variable>
     <a><xsl:value-of select="replace($t, $p, '[$11]($25)')"/></a>
  </xsl:template>

</xsl:stylesheet>